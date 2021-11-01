// Package compressor provides compressor implementation, which can be embedded into other programs, while
// keeping the core functionality intact, like reading the configuration files etc.
//
// This package responsibility:
// - Parse flags into more structured form.
// - Read and parse configuration files for configuring running action.
// - Load values from environment variables.
// - Setup dependencies for actions (e.g. construct client).
// - Finally, run requested action.
package compressor

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"sigs.k8s.io/yaml"

	"github.com/invidian/golang-cli-testing-example/pkg/compressor"
)

const (
	// ActionCompress ...
	ActionCompress = "compress"
	// ActionDecompress ...
	ActionDecompress = "decompress"
	// FormatEnv ...
	FormatEnv = "COMPRESSOR_FORMAT"

	// DefaultConfigPath ...
	DefaultConfigPath = "config.yaml"
)

// Config ...
type Config struct {
	Format string `json:"format"`
}

// Cli ...
type Cli struct {
	// Args are usually os.Args.
	Args []string

	// Output is usually stdout where user messages will be printed, but user may also request compression
	// result to be printed directly into it as well.
	Output io.Writer

	// Output is usually stderr where error messages will be printed e.g. separately from output data.
	ErrorOutput io.Writer

	// Input is usually stdin for direct user input.
	Input io.Reader

	action     string
	format     string
	configPath string
	inputPath  string
}

// Run ...
func (c *Cli) Run(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return fmt.Errorf("validating CLI configuration: %w", err)
	}

	c.format = os.Getenv(FormatEnv)
	c.configPath = DefaultConfigPath

	// Parse arguments.
	if err := c.parseArgs(); err != nil {
		return fmt.Errorf("parsing arguments: %w", err)
	}

	switch c.action {
	case "help":
		fmt.Fprintln(c.Output, usage())

		return nil
	case ActionCompress, ActionDecompress:
		return c.runAction(ctx)
	}

	fmt.Fprintln(c.ErrorOutput, usage())

	return fmt.Errorf("no action specified")
}

func (c *Cli) runAction(ctx context.Context) error {
	if err := c.readConfig(); err != nil {
		return fmt.Errorf("reading configuration: %w", err)
	}

	input, err := c.selectUserInput(c.Input)
	if err != nil {
		return fmt.Errorf("selecting user input: %w", err)
	}

	config := compressor.Config{
		Format: compressor.Format(c.format),
	}

	client, err := compressor.NewClient(config)
	if err != nil {
		return fmt.Errorf("creating compressor client: %w", err)
	}

	var output io.Reader

	var errCh chan error

	switch c.action {
	case ActionCompress:
		output, errCh = client.Compress(ctx, input)
	case ActionDecompress:
		output, errCh = client.Decompress(ctx, input)
	}

	if _, err := io.Copy(c.Output, output); err != nil {
		return fmt.Errorf("copying action output: %w", err)
	}

	if err := <-errCh; err != nil {
		return fmt.Errorf("running action: %w", err)
	}

	return nil
}

func (c *Cli) readConfig() error {
	configRaw, err := os.ReadFile(c.configPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading configuration file %q: %w", c.configPath, err)
	}

	config := &Config{}

	if err := yaml.Unmarshal(configRaw, config); err != nil {
		return fmt.Errorf("decoding config from file %q: %w", c.configPath, err)
	}

	if c.format == "" {
		c.format = config.Format
	}

	return nil
}

func (c *Cli) selectUserInput(userInput io.Reader) (io.Reader, error) {
	input := userInput

	var err error

	if c.inputPath == "" && input == nil {
		return nil, fmt.Errorf("either input or input path must be defined")
	}

	if c.inputPath != "" {
		input, err = os.Open(c.inputPath)
		if err != nil {
			return nil, fmt.Errorf("opening input file %q: %w", c.inputPath, err)
		}
	}

	return input, nil
}

func (c *Cli) parseArgs() error {
	for _, arg := range c.Args[1:] {
		switch arg {
		case "--help":
			c.action = "help"

			return nil
		case ActionCompress, ActionDecompress:
			if c.action != "" {
				return fmt.Errorf("action already specified")
			}

			c.action = arg
		default:
			if c.parseValueArgs(arg) {
				continue
			}

			fmt.Fprintln(c.ErrorOutput, usage())

			return fmt.Errorf("unknown argument %q: %v", arg, c.Args)
		}
	}

	return nil
}

func (c *Cli) parseValueArgs(arg string) bool {
	for flag, target := range map[string]*string{
		"format": &c.format,
		"config": &c.configPath,
		"input":  &c.inputPath,
	} {
		if parseStringArg(arg, flag, target) {
			return true
		}
	}

	return false
}

func parseStringArg(argument, flag string, destination *string) bool {
	flagFull := fmt.Sprintf("--%s", flag)
	if !strings.HasPrefix(argument, flagFull+"=") {
		return false
	}

	*destination = strings.Split(argument, "=")[1]

	return true
}

func (c *Cli) validate() error {
	if c.Output == nil {
		return fmt.Errorf("no output defined")
	}

	if c.ErrorOutput == nil {
		return fmt.Errorf("no error output defined")
	}

	if len(c.Args) == 0 {
		return fmt.Errorf("at least one argument must be provided")
	}

	return nil
}

func usage() string {
	return fmt.Sprintf(`Usage:
  %s [command]

Available Commands:
  compress   Compress data from standard input
  decompress Decompress data from standard input

Flags:
  --help   Help for %s.
  --format Specified compression format. Valid values are: %s. Default is %s.
  --config Path to optional configuration file. Default is %s.
  --input  Path to input file which should processed.`,
		os.Args[0], os.Args[0], strings.Join(compressor.AvailableFormats(), ", "),
		compressor.DefaultFormat, DefaultConfigPath)
}

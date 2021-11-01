package compressor_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/invidian/golang-cli-testing-example/cli/compressor"
	"github.com/invidian/golang-cli-testing-example/internal/testutil"
	pkgCompressor "github.com/invidian/golang-cli-testing-example/pkg/compressor"
)

func Test_Running_CLI_writes_resulting_data_into_given_output(t *testing.T) {
	t.Parallel()

	output := &bytes.Buffer{}
	errorOutput := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, compressor.ActionCompress},
		Output:      output,
		ErrorOutput: errorOutput,
		Input:       bytes.NewBufferString(testData),
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	if errorMessage := errorOutput.String(); len(errorMessage) != 0 {
		t.Fatalf("Unexpected error message printed to error output:\n%s", errorMessage)
	}

	if output := output.String(); len(output) == 0 {
		t.Fatalf("Expected some output to be printed")
	}
}

func Test_Running_CLI_reads_input_from_requested_input_file(t *testing.T) {
	t.Parallel()

	expectedOutput := testData

	inputPath := filepath.Join(t.TempDir(), "input")

	if err := os.WriteFile(inputPath, []byte(expectedOutput), 0o600); err != nil {
		t.Fatalf("Failed input file: %v", err)
	}

	output := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, compressor.ActionCompress, "--format=noop", "--input=" + inputPath},
		Output:      output,
		ErrorOutput: &bytes.Buffer{},
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	if gotOutput := output.String(); gotOutput != expectedOutput {
		t.Fatalf("Expected to get output %q, got %q", expectedOutput, gotOutput)
	}
}

//nolint:paralleltest // No parallelization as we tinker with working directory here which is global.
func Test_Running_CLI_tries_reading_settings_from_default_configuration_file(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(configPath, []byte("format: noop"), 0o600); err != nil {
		t.Fatalf("Failed writing configuration file: %v", err)
	}

	oldWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Error getting working directory: %v", err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed changing working directory to %q: %v", dir, err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldWorkingDir); err != nil {
			t.Fatalf("Failed restoring working directory: %v", err)
		}
	})

	expectedOutput := testData

	output := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, compressor.ActionCompress},
		Output:      output,
		ErrorOutput: &bytes.Buffer{},
		Input:       bytes.NewBufferString(expectedOutput),
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	if gotOutput := output.String(); gotOutput != expectedOutput {
		t.Fatalf("Expected to get output %q, got %q", expectedOutput, gotOutput)
	}
}

func Test_Running_CLI_reads_format_setting_from_specified_configuration_file_when_requested(t *testing.T) {
	t.Parallel()

	configPath := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(configPath, []byte("format: noop"), 0o600); err != nil {
		t.Fatalf("Failed writing configuration file: %v", err)
	}

	expectedOutput := testData

	output := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, compressor.ActionCompress, "--config=" + configPath},
		Output:      output,
		ErrorOutput: &bytes.Buffer{},
		Input:       bytes.NewBufferString(expectedOutput),
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	if gotOutput := output.String(); gotOutput != expectedOutput {
		t.Fatalf("Expected to get output %q, got %q", expectedOutput, gotOutput)
	}
}

func Test_Running_CLI_use_specified_format_for_actions(t *testing.T) {
	t.Parallel()

	expectedOutput := testData

	output := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, compressor.ActionCompress, "--format=noop"},
		Output:      output,
		ErrorOutput: &bytes.Buffer{},
		Input:       bytes.NewBufferString(expectedOutput),
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	if gotOutput := output.String(); gotOutput != expectedOutput {
		t.Fatalf("Expected to get output %q, got %q", expectedOutput, gotOutput)
	}
}

//nolint:paralleltest // This test sets environment variables.
func Test_Running_CLI_reads_default_format_from_environment_variable(t *testing.T) {
	t.Setenv(compressor.FormatEnv, "noop")

	expectedOutput := testData

	output := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, compressor.ActionCompress},
		Output:      output,
		ErrorOutput: &bytes.Buffer{},
		Input:       bytes.NewBufferString(expectedOutput),
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	if gotOutput := output.String(); gotOutput != expectedOutput {
		t.Fatalf("Expected to get output %q, got %q", expectedOutput, gotOutput)
	}
}

//nolint:paralleltest // This test sets environment variables.
func Test_Running_CLI_prefers_format_setting_from_arguments_over_environment_variable(t *testing.T) {
	t.Setenv(compressor.FormatEnv, string(pkgCompressor.FormatGzip))

	expectedOutput := testData

	output := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, compressor.ActionCompress, "--format=" + string(pkgCompressor.FormatNoop)},
		Output:      output,
		ErrorOutput: &bytes.Buffer{},
		Input:       bytes.NewBufferString(expectedOutput),
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	if gotOutput := output.String(); gotOutput != expectedOutput {
		t.Fatalf("Expected to get output %q, got %q", expectedOutput, gotOutput)
	}
}

func Test_Running_CLI_when_requested_help_via_flag_returns_no_error(t *testing.T) {
	t.Parallel()

	output := &bytes.Buffer{}
	errorOutput := &bytes.Buffer{}

	cli := compressor.Cli{
		Args:        []string{testCommand, "--help"},
		Output:      output,
		ErrorOutput: errorOutput,
		Input:       &bytes.Buffer{},
	}

	if err := cli.Run(testutil.ContextWithDeadline(t)); err != nil {
		t.Fatalf("Unexpected error running CLI: %v", err)
	}

	t.Run("and_prints_usage_message_with_binary_name_to_configured_output", func(t *testing.T) {
		t.Parallel()

		expectedOutput := "Usage:"
		if output := output.String(); !strings.Contains(output, expectedOutput) {
			t.Fatalf("Expected output to include %q, got:\n%s", expectedOutput, output)
		}
	})

	t.Run("and_does_not_print_any_error_message", func(t *testing.T) {
		t.Parallel()

		if errorMessage := errorOutput.String(); len(errorMessage) != 0 {
			t.Fatalf("Didn't expect error output, got:\n%s", errorMessage)
		}
	})
}

//nolint:funlen,gocognit,cyclop // Just many isolated test-cases.
func Test_Running_CLI_returns_error_when(t *testing.T) {
	t.Parallel()

	t.Run("output_is_not_set", func(t *testing.T) {
		t.Parallel()

		cli := compressor.Cli{
			Args:        []string{testCommand, "--help"},
			Output:      nil,
			ErrorOutput: &bytes.Buffer{},
			Input:       &bytes.Buffer{},
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected CLI run error")
		}
	})

	t.Run("error_output_is_not_set", func(t *testing.T) {
		t.Parallel()

		cli := compressor.Cli{
			Args:        []string{testCommand, "--help"},
			Output:      &bytes.Buffer{},
			ErrorOutput: nil,
			Input:       &bytes.Buffer{},
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected CLI run error")
		}
	})

	t.Run("neither_input_and_input_flag_are_not_set", func(t *testing.T) {
		t.Parallel()

		cli := compressor.Cli{
			Args:        []string{testCommand, "compress"},
			Output:      &bytes.Buffer{},
			ErrorOutput: &bytes.Buffer{},
			Input:       nil,
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected CLI run error")
		}
	})

	t.Run("no_arguments_are_given", func(t *testing.T) {
		t.Parallel()

		cli := compressor.Cli{
			Args:        []string{},
			Output:      &bytes.Buffer{},
			ErrorOutput: &bytes.Buffer{},
			Input:       &bytes.Buffer{},
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}
	})

	t.Run("no_action_argument_is_specified", func(t *testing.T) {
		t.Parallel()

		errorOutput := &bytes.Buffer{}

		cli := compressor.Cli{
			Args:        []string{testCommand},
			Output:      &bytes.Buffer{},
			ErrorOutput: errorOutput,
			Input:       &bytes.Buffer{},
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}

		if len(errorOutput.String()) == 0 {
			t.Fatalf("Expected error message to be printed to error output")
		}
	})

	t.Run("trying_to_run_unknown_action", func(t *testing.T) {
		t.Parallel()

		errorOutput := &bytes.Buffer{}

		cli := compressor.Cli{
			Args:        []string{testCommand, "unknown-action"},
			Output:      &bytes.Buffer{},
			ErrorOutput: errorOutput,
			Input:       &bytes.Buffer{},
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}

		if len(errorOutput.String()) == 0 {
			t.Fatalf("Expected error message to be printed to error output")
		}

		t.Run("and_prints_usage_message_only_once", func(t *testing.T) {
			t.Parallel()

			if strings.Count(errorOutput.String(), "Usage:") > 1 {
				t.Fatalf("Expected usage message to be printed only once, got:\n%s", errorOutput.String())
			}
		})
	})

	t.Run("multiple_valid_actions_are_specified", func(t *testing.T) {
		t.Parallel()

		cli := compressor.Cli{
			Args:        []string{testCommand, compressor.ActionDecompress, compressor.ActionCompress},
			Output:      &bytes.Buffer{},
			ErrorOutput: &bytes.Buffer{},
			Input:       bytes.NewBufferString(testData),
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}
	})
	t.Run("creating_compression_client_fails", func(t *testing.T) {
		t.Parallel()

		cli := compressor.Cli{
			Args:        []string{testCommand, compressor.ActionCompress, "--format=foo"},
			Output:      &bytes.Buffer{},
			ErrorOutput: &bytes.Buffer{},
			Input:       bytes.NewBufferString(testData),
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}
	})

	t.Run("requested_input_file_does_not_exit", func(t *testing.T) {
		t.Parallel()

		inputPath := filepath.Join(t.TempDir(), "nonexisting")

		output := &bytes.Buffer{}

		cli := compressor.Cli{
			Args:        []string{testCommand, compressor.ActionCompress, "--input=" + inputPath},
			Output:      output,
			ErrorOutput: &bytes.Buffer{},
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}
	})

	t.Run("requested_input_file_is_not_readable", func(t *testing.T) {
		t.Parallel()

		expectedOutput := testData

		inputPath := filepath.Join(t.TempDir(), "input")

		if err := os.WriteFile(inputPath, []byte(expectedOutput), 0o000); err != nil {
			t.Fatalf("Failed input file: %v", err)
		}

		output := &bytes.Buffer{}

		cli := compressor.Cli{
			Args:        []string{testCommand, compressor.ActionCompress, "--input=" + inputPath},
			Output:      output,
			ErrorOutput: &bytes.Buffer{},
		}

		err := cli.Run(testutil.ContextWithDeadline(t))
		if err == nil {
			t.Fatalf("Expected error running CLI")
		}

		if !errors.Is(err, os.ErrPermission) {
			t.Fatalf("Unexpected error %q, expected %q", err, os.ErrPermission)
		}
	})

	t.Run("configuration_file_exists_but_it_is_not_readable", func(t *testing.T) {
		t.Parallel()

		configPath := filepath.Join(t.TempDir(), "config.yaml")

		if err := os.WriteFile(configPath, []byte("format: noop"), 0o000); err != nil {
			t.Fatalf("Failed writing configuration file: %v", err)
		}

		expectedOutput := testData

		output := &bytes.Buffer{}

		cli := compressor.Cli{
			Args:        []string{testCommand, compressor.ActionCompress, "--config=" + configPath},
			Output:      output,
			ErrorOutput: &bytes.Buffer{},
			Input:       bytes.NewBufferString(expectedOutput),
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}
	})

	t.Run("configuration_file_is_not_a_valid_YAML", func(t *testing.T) {
		t.Parallel()

		configPath := filepath.Join(t.TempDir(), "config.yaml")

		if err := os.WriteFile(configPath, []byte("format"), 0o600); err != nil {
			t.Fatalf("Failed writing configuration file: %v", err)
		}

		expectedOutput := testData

		output := &bytes.Buffer{}

		cli := compressor.Cli{
			Args:        []string{testCommand, compressor.ActionCompress, "--config=" + configPath},
			Output:      output,
			ErrorOutput: &bytes.Buffer{},
			Input:       bytes.NewBufferString(expectedOutput),
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}
	})

	t.Run("action_fails", func(t *testing.T) {
		t.Parallel()

		cli := compressor.Cli{
			Args:        []string{testCommand, compressor.ActionDecompress},
			Output:      &bytes.Buffer{},
			ErrorOutput: &bytes.Buffer{},
			Input:       bytes.NewBufferString(testData),
		}

		if err := cli.Run(testutil.ContextWithDeadline(t)); err == nil {
			t.Fatalf("Expected error running CLI")
		}
	})

	t.Run("writing_to_given_output_fails", func(t *testing.T) {
		t.Parallel()

		expectedError := fmt.Errorf("sample")

		cli := compressor.Cli{
			Args: []string{testCommand, compressor.ActionCompress},
			Output: &testFailingWriter{
				err: expectedError,
			},
			ErrorOutput: &bytes.Buffer{},
			Input:       bytes.NewBufferString(testData),
		}

		err := cli.Run(testutil.ContextWithDeadline(t))
		if err == nil {
			t.Fatalf("Expected error running CLI")
		}

		if !errors.Is(err, expectedError) {
			t.Fatalf("Expected error %q, got %q", expectedError, err)
		}
	})
}

func TestMain(m *testing.M) {
	// Ensure user has no format environment variable set when running tests, to make sure
	// test results are not affected by user environment.
	if err := os.Unsetenv(compressor.FormatEnv); err != nil {
		fmt.Fprintf(os.Stderr, "Failed unsetting environment variable %q: %v", compressor.FormatEnv, err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

type testFailingWriter struct {
	err error
}

func (f *testFailingWriter) Write([]byte) (int, error) {
	return 0, f.err
}

const (
	testCommand = "testCommand"
	testData    = "testData"
)

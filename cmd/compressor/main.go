// Package main contains an actual entrypoint for CLI code, but it's only purpose is to provide an integration
// point for all dependencies of CLI, like input and output streams, CLI arguments, signal handling etc, so
// components, which are not useful when you try to embed the CLI code into some other application.
//
// Global resources accessed by the CLI:
// - stdin (os.Stdin)
// - stderr (os.Stderr)
// - stdout (os.Stdout)
// - environment variables (os.Environ())
// - command line arguments (os.Args)
// - process exit code
// - process signals
//
// In another words, all listed above is sort of I/O, which by design should be pluggable.
//
// This package responsibilities:
// - Converting CLI errors into appropriate exit codes.
// - Handling OS signals to context/control channel conversion.
//
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/invidian/golang-cli-testing-example/cli/compressor"
)

func main() {
	os.Exit(run())
}

func run() int {
	cli := compressor.Cli{
		Output:      os.Stdout,
		Input:       os.Stdin,
		ErrorOutput: os.Stderr,
		Args:        os.Args,
	}

	if err := cli.Run(signalContext()); err != nil {
		fmt.Fprintf(os.Stderr, "Error running CLI: %v\n", err)

		return 1
	}

	return 0
}

func signalContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		cancel()
	}()

	return ctx
}

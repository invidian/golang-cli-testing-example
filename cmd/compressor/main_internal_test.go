package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func Test_Main_exits_with_exit_code(t *testing.T) {
	t.Parallel()

	t.Run("0_when_CLI_returns_no_error", func(t *testing.T) {
		t.Parallel()

		cmd := testCmd(helpFlag)
		if err := cmd.Run(); err != nil {
			t.Fatalf("Unexpected error running command: %v", err)
		}

		expectedExitCode := 0

		if exitCode := cmd.ProcessState.ExitCode(); exitCode != expectedExitCode {
			t.Fatalf("Expected exit code %d, got %d", expectedExitCode, exitCode)
		}
	})

	t.Run("1_when_CLI_returns_error", func(t *testing.T) {
		t.Parallel()

		err := testCmd().Run()
		if err == nil {
			t.Fatalf("Expected error running command")
		}

		var exitErr *exec.ExitError

		if !errors.As(err, &exitErr) {
			t.Fatalf("Expected to get ExitError, got %t", err)
		}

		expectedExitCode := 1

		if exitCode := exitErr.ExitCode(); exitCode != expectedExitCode {
			t.Fatalf("Expected exit code %d, got %d", expectedExitCode, exitCode)
		}
	})
}

func Test_Main_prints_regular_output_to_stdout(t *testing.T) {
	t.Parallel()

	output, err := testCmd(helpFlag).Output()
	if err != nil {
		t.Fatalf("Unexpected error running command: %v", err)
	}

	if string(output) == "" {
		t.Fatalf("Expected something to be printed to stdout")
	}
}

func Test_Main_prints_error_messages_to_stderr(t *testing.T) {
	t.Parallel()

	cmd := testCmd()
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	if err := cmd.Run(); err == nil {
		t.Fatalf("Expected error running command")
	}

	errOutput := stderr.String()

	if len(errOutput) == 0 {
		t.Fatalf("Expected some stderr output")
	}

	expectedErrorMessage := "Error running CLI"

	if !strings.Contains(errOutput, expectedErrorMessage) {
		t.Fatalf("Expected error message to include %q, got:\n%s", expectedErrorMessage, errOutput)
	}
}

//nolint:funlen,cyclop // Just long test.
func Test_Main_handles_gracefully(t *testing.T) {
	t.Parallel()

	for name, signal := range map[string]os.Signal{
		"interrupt_signal":   os.Interrupt,
		"termination_signal": syscall.SIGTERM,
	} {
		signal := signal

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			cmd := testCmd("--input=/dev/zero", "compress")
			stderr := &bytes.Buffer{}
			cmd.Stderr = stderr

			errCh := make(chan error, 1)

			if err := cmd.Start(); err != nil {
				t.Fatalf("Failed starting process: %v", err)
			}

			// Force-trigger killing the process after test ends to avoid leaving orphan processes
			// in case the test fails.
			t.Cleanup(func() {
				if err := cmd.Process.Kill(); err != nil {
					t.Logf("Failed killing process: %v", err)
				}
			})

			go func() {
				errCh <- cmd.Wait()
			}()

			// Ensure process did not exit prematurely.
			testStartupTime := time.NewTimer(100 * time.Millisecond)

			select {
			case err := <-errCh:
				t.Fatalf("Process should still be running, got: %v", err)
			case <-testStartupTime.C:
			}

			if err := cmd.Process.Signal(signal); err != nil {
				t.Fatalf("Sending signal to process failed: %v", err)
			}

			timeout := time.NewTimer(time.Second)

			expectedError := "context canceled"

			select {
			case err := <-errCh:
				var exitErr *exec.ExitError

				if !errors.As(err, &exitErr) {
					t.Fatalf("Expected to get ExitError, got %t", err)
				}

				expectedExitCode := 1

				if exitCode := exitErr.ExitCode(); exitCode != expectedExitCode {
					t.Fatalf("Expected exit code %d, got %d", expectedExitCode, exitCode)
				}

				if !strings.HasSuffix(strings.TrimSpace(stderr.String()), expectedError) {
					t.Fatalf("Expected error output to end with %q, got:\n%s", expectedError, stderr.String())
				}
			case <-timeout.C:
				t.Fatal("Compression did not stop within expected timeout")
			}
		})
	}
}

const (
	testFlagMain = "-test.main"
	helpFlag     = "--help"
)

// This wrapper function allows running main() with desired arguments from the unit tests.
func TestMain(m *testing.M) {
	for i, arg := range os.Args {
		if arg == testFlagMain {
			os.Args = append(os.Args[:i], os.Args[i+1:]...)

			main()

			return
		}
	}

	os.Exit(m.Run())
}

func testCmd(args ...string) *exec.Cmd {
	return exec.Command(os.Args[0], append([]string{testFlagMain}, args...)...)
}

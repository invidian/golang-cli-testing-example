package compressor_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/invidian/golang-cli-testing-example/internal/testutil"
	"github.com/invidian/golang-cli-testing-example/pkg/compressor"
)

func Test_Compressing_and_decompressing_data_using_same_compressor_multiple_times_restores_original_data(t *testing.T) {
	t.Parallel()

	client, err := compressor.NewClient()
	if err != nil {
		t.Fatalf("Unexpected error creating client: %v", err)
	}

	ctx := testutil.ContextWithDeadline(t)

	// Run multiple times in parallel to make sure there is no global buffers shared.
	tries := 5

	errCh := make(chan error, tries)

	for i := 0; i < tries; i++ {
		go func() {
			errCh <- func() error {
				data := make([]byte, 0, 128)
				if _, err := rand.Read(data); err != nil {
					return fmt.Errorf("generating random data for compression: %w", err)
				}

				compressedData, compressErrCh := client.Compress(ctx, bytes.NewBuffer(data))

				reader, decompressErrCh := client.Decompress(ctx, io.NopCloser(compressedData))

				decompressedData, err := io.ReadAll(reader)
				if err != nil {
					return fmt.Errorf("reading decompressed data: %w", err)
				}

				if err := <-compressErrCh; err != nil {
					return fmt.Errorf("compressing: %w", err)
				}

				if err := <-decompressErrCh; err != nil {
					return fmt.Errorf("decompressing: %w", err)
				}

				if string(decompressedData) != string(data) {
					return fmt.Errorf("expected decompressed data to be %q, got %q", data, string(decompressedData))
				}

				return nil
			}()
		}()
	}

	for i := 0; i < tries; i++ {
		if err := <-errCh; err != nil {
			t.Errorf("Error in attempt %d: %v", i, err)
		}
	}
}

func Test_Compressor_use_gzip_format_for_compression(t *testing.T) {
	t.Parallel()

	for name, testConfig := range map[string]*compressor.Config{
		"by_default": nil,
		"when_configured_explicitly": {
			Format: compressor.FormatGzip,
		},
		"with_empty_config": {},
	} {
		testConfig := testConfig

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client, err := compressor.NewClient()
			if testConfig != nil {
				client, err = compressor.NewClient(*testConfig)
			}

			if err != nil {
				t.Fatalf("Unexpected error creating client: %v", err)
			}

			compressedData, errCh := client.Compress(testutil.ContextWithDeadline(t), bytes.NewBufferString(testData))

			reader, err := gzip.NewReader(compressedData)
			if err != nil {
				t.Errorf("Failed creating gzip reader: %v", err)
			}

			decompressedData, err := io.ReadAll(reader)
			if err != nil {
				t.Errorf("Failed decompressing data: %v", err)
			}

			if err := <-errCh; err != nil {
				t.Fatalf("Unexpected compression error: %v", err)
			}

			if string(decompressedData) != testData {
				t.Fatalf("Expected decompressed data to be %q, got %q", testData, string(decompressedData))
			}
		})
	}
}

func Test_Compressor_supports_noop_compression_format_which_does_not_modify_data(t *testing.T) {
	t.Parallel()

	config := compressor.Config{
		Format: compressor.FormatNoop,
	}

	client, err := compressor.NewClient(config)
	if err != nil {
		t.Fatalf("Unexpected error creating client: %v", err)
	}

	ctx := testutil.ContextWithDeadline(t)
	compressedData, errCh := client.Compress(ctx, bytes.NewBufferString(testData))

	decompressedDataReader, decompressionErrCh := client.Decompress(ctx, compressedData)

	decompressedData, err := io.ReadAll(decompressedDataReader)
	if err != nil {
		t.Errorf("Failed decompressing data: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Unexpected compression error: %v", err)
	}

	if err := <-decompressionErrCh; err != nil {
		t.Fatalf("Unexpected decompression error: %v", err)
	}

	if string(decompressedData) != testData {
		t.Fatalf("Expected decompressed data to be %q, got %q", testData, string(decompressedData))
	}
}

//nolint:funlen // Just many test cases.
func Test_Creating_compressor_returns_error_when(t *testing.T) {
	t.Parallel()

	t.Run("unknown_compression_format_is_requested", func(t *testing.T) {
		t.Parallel()

		badFormat := "badFormat"

		config := compressor.Config{
			Format: compressor.Format(badFormat),
		}

		client, err := compressor.NewClient(config)
		if err == nil {
			t.Fatalf("Expected client creating error")
		}

		if !strings.Contains(err.Error(), badFormat) {
			t.Fatalf("Expected error message to contain %q, got %v", badFormat, err)
		}

		if client != nil {
			t.Fatalf("When creating client returns error, no client should be returned")
		}
	})

	t.Run("multiple_configs_are_passed", func(t *testing.T) {
		t.Parallel()

		c, err := compressor.NewClient(compressor.Config{}, compressor.Config{})
		if err == nil {
			t.Fatalf("Expected client creating error")
		}

		if c != nil {
			t.Fatalf("When creating client returns error, no client should be returned")
		}
	})

	t.Run("compressor_is_configured_without_decompressor", func(t *testing.T) {
		t.Parallel()

		config := compressor.Config{
			Compressor: nopCompressor,
		}

		c, err := compressor.NewClient(config)
		if err == nil {
			t.Fatalf("Expected client creating error")
		}

		if c != nil {
			t.Fatalf("When creating client returns error, no client should be returned")
		}
	})

	t.Run("decompressor_is_configured_without_compressor", func(t *testing.T) {
		t.Parallel()

		config := compressor.Config{
			Decompressor: nopDecompressor,
		}

		c, err := compressor.NewClient(config)
		if err == nil {
			t.Fatalf("Expected client creating error")
		}

		if c != nil {
			t.Fatalf("When creating client returns error, no client should be returned")
		}
	})
}

func Test_Compressor_supports_all_available_formats(t *testing.T) {
	t.Parallel()

	for _, format := range compressor.AvailableFormats() {
		config := compressor.Config{
			Format: compressor.Format(format),
		}

		if _, err := compressor.NewClient(config); err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}
	}
}

func Test_Compressor_use_gzip_format_for_decompressor_by_default(t *testing.T) {
	t.Parallel()

	client, err := compressor.NewClient()
	if err != nil {
		t.Fatalf("Unexpected error creating client: %v", err)
	}

	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	if _, err := writer.Write([]byte(testData)); err != nil {
		t.Fatalf("Failed writing data to compress: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Failed closing writer: %v", err)
	}

	reader, errCh := client.Decompress(testutil.ContextWithDeadline(t), io.NopCloser(&buf))

	decompressedData, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Failed decompressing data: %v", err)
	}

	if err := <-errCh; err != nil {
		t.Fatalf("Unexpected compression error: %v", err)
	}

	if string(decompressedData) != testData {
		t.Fatalf("Expected decompressed data to be %q, got %q", testData, string(decompressedData))
	}
}

//nolint:funlen,cyclop // Just many test cases.
func Test_Compressor_when_given_context_is_cancelled(t *testing.T) {
	t.Parallel()

	t.Run("stops_compression", func(t *testing.T) {
		t.Parallel()

		client, err := compressor.NewClient()
		if err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}

		timeout := time.NewTimer(time.Second)

		ctx, cancel := context.WithTimeout(testutil.ContextWithDeadline(t), 200*time.Millisecond)
		defer cancel()

		_, errCh := client.Compress(ctx, rand.New(rand.NewSource(time.Now().UnixNano())))

		select {
		case err := <-errCh:
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Unexpected compression error: %v", err)
			}
		case <-timeout.C:
			t.Fatal("Compression did not stop within expected timeout")
		}
	})

	t.Run("stops_decompression", func(t *testing.T) {
		t.Parallel()

		client, err := compressor.NewClient()
		if err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}

		ctx, cancel := context.WithCancel(testutil.ContextWithDeadline(t))
		defer cancel()

		compressedData, compressErrCh := client.Compress(ctx, rand.New(rand.NewSource(time.Now().UnixNano())))

		ctx, cancelTimeout := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancelTimeout()

		reader, decompressErrCh := client.Decompress(ctx, io.NopCloser(compressedData))

		errCh := make(chan error, 1)

		go func() {
			errCh <- func() error {
				if _, err := io.ReadAll(reader); err != nil {
					return fmt.Errorf("reading decompressed data: %w", err)
				}

				if err := <-compressErrCh; err != nil {
					return fmt.Errorf("compressing: %w", err)
				}

				return nil
			}()
		}()

		timeout := time.NewTimer(time.Second)

		select {
		case err := <-decompressErrCh:
			if err != nil && !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("Unexpected compression error: %v", err)
			}
		case <-timeout.C:
			t.Fatal("Compression did not stop within expected timeout")
		}

		cancel()

		if err := <-errCh; err != nil && !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Unexpected client error: %v", err)
		}
	})
}

//nolint:funlen // Just many test cases.
func Test_Compression_returns_error_when(t *testing.T) {
	t.Parallel()

	t.Run("closing_compressor_fails", func(t *testing.T) {
		t.Parallel()

		expectedErr := fmt.Errorf("test error")

		config := compressor.Config{
			Compressor: func(wrc io.WriteCloser) io.WriteCloser {
				return &testReadWriteCloser{
					writeF: func(b []byte) (int, error) {
						//nolint:wrapcheck // We don't care about error wrapping in test code.
						return wrc.Write(b)
					},
					closeF: func() error {
						if err := wrc.Close(); err != nil {
							t.Logf("Failed closing test writer: %v", err)
						}

						return expectedErr
					},
				}
			},
			Decompressor: nopDecompressor,
		}

		c, err := compressor.NewClient(config)
		if err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}

		output, errCh := c.Compress(testutil.ContextWithDeadline(t), bytes.NewBufferString(testData))

		if _, err := io.Copy(io.Discard, output); err != nil {
			t.Fatalf("Failed discarding compression output: %v", err)
		}

		if err := <-errCh; !errors.Is(err, expectedErr) {
			t.Fatalf("Expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("bad_input_is_given", func(t *testing.T) {
		t.Parallel()

		config := compressor.Config{
			Compressor: func(wrc io.WriteCloser) io.WriteCloser {
				return &testReadWriteCloser{
					writeF: func(b []byte) (int, error) {
						//nolint:wrapcheck // We don't care about error wrapping in test code.
						return wrc.Write(b)
					},
					closeF: func() error {
						//nolint:wrapcheck // We don't care about error wrapping in test code.
						return wrc.Close()
					},
				}
			},
			Decompressor: nopDecompressor,
		}

		client, err := compressor.NewClient(config)
		if err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}

		badReader := &testReadWriteCloser{
			readF: func(p []byte) (n int, err error) {
				return 0, fmt.Errorf("reading error")
			},
		}

		_, errCh := client.Compress(testutil.ContextWithDeadline(t), badReader)

		if err := <-errCh; err == nil {
			t.Fatalf("Expected error")
		}
	})
}

//nolint:funlen // Just many test cases.
func Test_Decompression_returns_error_when(t *testing.T) {
	t.Parallel()
	t.Run("creating_decompressor_fails", func(t *testing.T) {
		t.Parallel()

		c, err := compressor.NewClient()
		if err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}

		_, errCh := c.Decompress(testutil.ContextWithDeadline(t), io.NopCloser(bytes.NewBufferString(testData)))

		err = <-errCh
		if err == nil {
			t.Fatalf("Expected compression error")
		}
	})

	t.Run("bad_input_is_given", func(t *testing.T) {
		t.Parallel()

		config := compressor.Config{
			Compressor:   nopCompressor,
			Decompressor: nopDecompressor,
		}

		client, err := compressor.NewClient(config)
		if err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}

		badReader := &testReadWriteCloser{
			readF: func(p []byte) (n int, err error) {
				return 0, fmt.Errorf("reading error")
			},
		}

		_, errCh := client.Decompress(testutil.ContextWithDeadline(t), badReader)

		if err := <-errCh; err == nil {
			t.Fatalf("Expected error")
		}
	})

	t.Run("closing_decompressor_fails", func(t *testing.T) {
		t.Parallel()

		expectedErr := fmt.Errorf("test error")

		config := compressor.Config{
			Compressor: nopCompressor,
			Decompressor: func(a io.Reader) (io.ReadCloser, error) {
				return &testReadWriteCloser{
					readF: func(b []byte) (int, error) {
						//nolint:wrapcheck // We don't care about error wrapping in test code.
						return a.Read(b)
					},
					closeF: func() error {
						return expectedErr
					},
				}, nil
			},
		}

		c, err := compressor.NewClient(config)
		if err != nil {
			t.Fatalf("Unexpected error creating client: %v", err)
		}

		output, errCh := c.Decompress(testutil.ContextWithDeadline(t), bytes.NewBufferString(testData))

		if _, err := io.Copy(io.Discard, output); err != nil {
			t.Fatalf("Failed discarding compression output: %v", err)
		}

		if err := <-errCh; !errors.Is(err, expectedErr) {
			t.Fatalf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

const testData = "foo"

func nopCompressor(a io.WriteCloser) io.WriteCloser {
	return a
}

func nopDecompressor(a io.Reader) (io.ReadCloser, error) {
	return io.NopCloser(a), nil
}

type testReadWriteCloser struct {
	writeF func([]byte) (int, error)
	closeF func() error
	readF  func(p []byte) (n int, err error)
}

func (trwc *testReadWriteCloser) Write(b []byte) (int, error) {
	return trwc.writeF(b)
}

func (trwc *testReadWriteCloser) Close() error {
	return trwc.closeF()
}

func (trwc *testReadWriteCloser) Read(p []byte) (n int, err error) {
	return trwc.readF(p)
}

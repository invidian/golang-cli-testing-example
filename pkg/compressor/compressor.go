// Package compressor provides simple compression tool with configurable compression format.
package compressor

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"

	"github.com/go-git/go-git/v5/utils/ioutil"
)

// Format ...
type Format string

const (
	// FormatGzip ...
	FormatGzip Format = "gzip"
	// FormatNoop ...
	FormatNoop Format = "noop"

	// DefaultFormat ...
	DefaultFormat = FormatGzip
)

// AvailableFormats ...
func AvailableFormats() []string {
	return []string{
		string(FormatGzip),
		string(FormatNoop),
	}
}

// Config ...
type Config struct {
	Format       Format
	Compressor   func(io.WriteCloser) io.WriteCloser
	Decompressor func(io.Reader) (io.ReadCloser, error)
}

// Client ...
type Client interface {
	Compress(context.Context, io.Reader) (io.Reader, chan error)
	Decompress(context.Context, io.Reader) (io.Reader, chan error)
}

type client struct {
	compressor   func(io.WriteCloser) io.WriteCloser
	decompressor func(io.Reader) (io.ReadCloser, error)
}

func (c Config) validate() error {
	if c.Decompressor == nil {
		return fmt.Errorf("decompressor must be configured")
	}

	if c.Compressor == nil {
		return fmt.Errorf("compressor must be configured")
	}

	return nil
}

// NewClient ...
func NewClient(configs ...Config) (Client, error) {
	if len(configs) > 1 {
		return nil, fmt.Errorf("only one config can be passed")
	}

	config := gzipConfig()

	if len(configs) == 1 {
		config = configs[0]
	}

	if config.Decompressor == nil && config.Compressor == nil {
		switch config.Format {
		case FormatGzip, "":
			config = gzipConfig()
		case FormatNoop:
			config = noopConfig()
		default:
			return nil, fmt.Errorf("unknown compression format %q", config.Format)
		}
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("validating configuration: %w", err)
	}

	return &client{
		compressor:   config.Compressor,
		decompressor: config.Decompressor,
	}, nil
}

// Compress ...
func (c *client) Compress(ctx context.Context, input io.Reader) (io.Reader, chan error) {
	compressedReader, compressedWriter := io.Pipe()

	ctxCompressedReader := ioutil.NewContextReader(ctx, compressedReader)
	ctxCompressedWriter := ioutil.NewContextWriteCloser(ctx, compressedWriter)

	errCh := make(chan error, 1)

	compressor := c.compressor(ctxCompressedWriter)

	go func() {
		errCh <- func() error {
			// Initialize compression by draining input.
			if _, err := io.Copy(compressor, input); err != nil {
				return fmt.Errorf("compressing data: %w", err)
			}

			// Ensure all data was flushed.
			if err := compressor.Close(); err != nil {
				return fmt.Errorf("closing compressor: %w", err)
			}

			// Close writing to pipe, so reading from it does not block infinitely.
			//
			//nolint:errcheck // Closing pipe always returns nil.
			ctxCompressedWriter.Close()

			return nil
		}()
	}()

	return ctxCompressedReader, errCh
}

// Decompress ...
func (c *client) Decompress(ctx context.Context, input io.Reader) (io.Reader, chan error) {
	decompressedReader, decompressedWriter := io.Pipe()

	ctxDecompressedReader := ioutil.NewContextReader(ctx, decompressedReader)
	ctxDecompressedWriter := ioutil.NewContextWriteCloser(ctx, decompressedWriter)

	errCh := make(chan error, 1)

	decompressor, err := c.decompressor(input)
	if err != nil {
		errCh <- fmt.Errorf("creating decompressor: %w", err)

		//nolint:errcheck // Closing pipe always returns nil.
		defer ctxDecompressedWriter.Close()

		return ctxDecompressedReader, errCh
	}

	go func() {
		errCh <- func() error {
			// Initialize decompression by draining input.
			if _, err := io.Copy(ctxDecompressedWriter, decompressor); err != nil {
				return fmt.Errorf("decompressing data: %w", err)
			}

			// Close writing to pipe, so reading from it does not block infinitely.
			//
			//nolint:errcheck // Closing pipe always returns nil.
			defer func() { _ = ctxDecompressedWriter.Close() }()

			// Ensure all data was flushed.
			if err := decompressor.Close(); err != nil {
				return fmt.Errorf("closing decompressor: %w", err)
			}

			return nil
		}()
	}()

	return ctxDecompressedReader, errCh
}

func gzipConfig() Config {
	return Config{
		Compressor: func(a io.WriteCloser) io.WriteCloser {
			return gzip.NewWriter(a)
		},
		Decompressor: func(a io.Reader) (io.ReadCloser, error) {
			rc, err := gzip.NewReader(a)
			if err != nil {
				return nil, fmt.Errorf("creating decompressor: %w", err)
			}

			return rc, nil
		},
	}
}

func noopConfig() Config {
	return Config{
		Compressor: func(a io.WriteCloser) io.WriteCloser {
			return a
		},
		Decompressor: func(a io.Reader) (io.ReadCloser, error) {
			return io.NopCloser(a), nil
		},
	}
}

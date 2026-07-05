// export_test.go exposes internal functions for use by the client_test package.
// This file is only compiled during tests.
package client

import (
	"io"
	"net/http"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// DoWithWriters is a test hook that calls the internal do() with explicit
// writers so tests can capture stdout/stderr without redirecting os.Stdout.
func DoWithWriters(opts *cli.Options, out, errOut io.Writer) error {
	return do(opts, out, errOut)
}

// ApplyHeader is a test hook that exposes the internal applyHeader function.
func ApplyHeader(req *http.Request, header string) error {
	return applyHeader(req, header)
}

// NewBodySource is a test hook that exposes the internal newBodySource function.
func NewBodySource(spec string) (*bodySource, error) {
	return newBodySource(spec)
}

// bodySource methods exposed for testing
// (The bodySource type itself needs to be exported)

// GetReader is a test hook for bodySource.getReader().
func (bs *bodySource) GetReader() (io.ReadCloser, error) {
	if bs == nil {
		return nil, nil
	}
	return bs.getReader()
}

// BufferStdinIfNeeded is a test hook for bodySource.bufferStdinIfNeeded().
func (bs *bodySource) BufferStdinIfNeeded() error {
	if bs == nil {
		return nil
	}
	return bs.bufferStdinIfNeeded()
}

// Close is a test hook for bodySource.close().
func (bs *bodySource) Close() error {
	if bs == nil {
		return nil
	}
	return bs.close()
}

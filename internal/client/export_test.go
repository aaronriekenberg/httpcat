// export_test.go exposes internal functions for testing.
package client

import (
	"io"
	"net/http"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// DoWithWriters calls do() with explicit writers for testing.
func DoWithWriters(opts *cli.Options, out, errOut io.Writer) error {
	return do(opts, out, errOut)
}

// ApplyHeader exposes the internal applyHeader function for testing.
func ApplyHeader(req *http.Request, header string) error {
	return applyHeader(req, header)
}

// NewBodySource exposes the internal newBodySource function for testing.
func NewBodySource(spec string) (*bodySource, error) {
	return newBodySource(spec)
}

// GetReader exposes the internal getReader method for testing.
func (bs *bodySource) GetReader() (io.ReadCloser, error) {
	return bs.getReader()
}

// BufferStdinIfNeeded exposes the internal bufferStdinIfNeeded method for testing.
func (bs *bodySource) BufferStdinIfNeeded() error {
	return bs.bufferStdinIfNeeded()
}

// Close exposes the internal close method for testing.
func (bs *bodySource) Close() error {
	return bs.close()
}

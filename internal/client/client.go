// Package client provides the HTTP client dispatcher for httpcat.
package client

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// newRequest creates an *http.Request from options.
func newRequest(opts *cli.Options) (*http.Request, error) {
	req, err := http.NewRequest(opts.Method, opts.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("building request: %w", err)
	}
	return req, nil
}

// Do dispatches the request to the appropriate HTTP implementation.
// --http3 / --http3-only  → HTTP/3 via quic-go (with fallback for --http3)
// --http2-prior-knowledge → HTTP/2 forced via golang.org/x/net/http2
// default                 → standard net/http (negotiates HTTP/1.1 or HTTP/2)
func Do(opts *cli.Options) error {
	return do(opts, os.Stdout, os.Stderr)
}

// do is the testable implementation.
func do(opts *cli.Options, out, errOut io.Writer) error {
	if opts.HTTP3 {
		err := doHTTP3(opts, out, errOut)
		if err == nil {
			return nil
		}
		if opts.HTTP3Only {
			return fmt.Errorf("HTTP/3 required but failed: %w", err)
		}
		// Fall back to HTTP/1.x or HTTP/2.
		return doHTTP12(opts, out, errOut)
	}
	return doHTTP12(opts, out, errOut)
}

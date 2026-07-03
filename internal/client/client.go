// Package client provides the HTTP client dispatcher for httpcat.
package client

import (
	"fmt"
	"net/http"

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
	if opts.HTTP3 {
		err := DoHTTP3(opts)
		if err == nil {
			return nil
		}
		if opts.HTTP3Only {
			return fmt.Errorf("HTTP/3 required but failed: %w", err)
		}
		// Fall back to HTTP/1.x or HTTP/2.
		return DoHTTP12(opts)
	}
	return DoHTTP12(opts)
}

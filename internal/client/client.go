// Package client provides the HTTP client dispatcher for httpcat.
package client

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// newRequest creates an *http.Request from options and applies custom headers.
func newRequest(opts *cli.Options) (*http.Request, error) {
	body, err := cli.ReadBody(opts)
	if err != nil {
		return nil, fmt.Errorf("reading body: %w", err)
	}

	req, err := http.NewRequest(opts.Method, opts.URL, body)
	if err != nil {
		if body != nil {
			body.Close()
		}
		return nil, fmt.Errorf("building request: %w", err)
	}

	// Apply custom headers
	for _, header := range opts.Headers {
		if err := applyHeader(req, header); err != nil {
			body.Close()
			return nil, err
		}
	}
	return req, nil
}

// applyHeader parses a header string ("Key: value") and sets it on the request.
func applyHeader(req *http.Request, header string) error {
	// Find the colon separator
	colonIdx := -1
	for i := 0; i < len(header); i++ {
		if header[i] == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx < 0 {
		return fmt.Errorf("invalid header format %q: must be 'Key: value'", header)
	}
	key := header[:colonIdx]
	value := header[colonIdx+1:]
	// Trim leading space from value
	for len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}
	if key == "" {
		return fmt.Errorf("invalid header format %q: key cannot be empty", header)
	}
	req.Header.Add(key, value)
	return nil
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

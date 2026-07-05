// Package client provides the HTTP client dispatcher for httpcat.
package client

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// newRequest creates an *http.Request with custom headers.
func newRequest(opts *cli.Options, bs *bodySource) (*http.Request, error) {
	// Get body reader
	var body io.ReadCloser
	bodySpec := opts.JSON
	if bodySpec == "" {
		bodySpec = opts.DataBinary
	}
	if bodySpec != "" {
		if bs == nil {
			return nil, fmt.Errorf("body source required")
		}
		var err error
		body, err = bs.getReader()
		if err != nil {
			return nil, fmt.Errorf("reading body: %w", err)
		}
	}

	// Create request
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

// applyHeader parses and sets a single header.
func applyHeader(req *http.Request, header string) error {
	idx := strings.IndexByte(header, ':')
	if idx < 0 {
		return fmt.Errorf("invalid header %q: missing colon", header)
	}

	key := header[:idx]
	if key == "" {
		return fmt.Errorf("invalid header %q: empty key", header)
	}

	// Trim leading space from value
	value := strings.TrimLeft(header[idx+1:], " ")
	req.Header.Add(key, value)
	return nil
}

// Do dispatches the request to the appropriate HTTP implementation.
func Do(opts *cli.Options) error {
	return do(opts, os.Stdout, os.Stderr)
}

// do is the testable implementation with custom writers.
func do(opts *cli.Options, out, errOut io.Writer) error {
	// Determine body spec
	bodySpec := opts.JSON
	if bodySpec == "" {
		bodySpec = opts.DataBinary
	}

	// Create body source
	bs, err := newBodySource(bodySpec)
	if err != nil {
		return fmt.Errorf("preparing body: %w", err)
	}
	defer bs.close()

	// Buffer stdin if HTTP/3 fallback is possible
	if opts.HTTP3 && !opts.HTTP3Only && strings.HasPrefix(bodySpec, "@-") {
		if err := bs.bufferStdinIfNeeded(); err != nil {
			return err
		}
	}

	// Try HTTP/3, then fallback
	if opts.HTTP3 {
		if err := doHTTP3(opts, out, errOut, bs); err == nil {
			return nil
		}
		if opts.HTTP3Only {
			return fmt.Errorf("HTTP/3 required but failed")
		}
	}

	return doHTTP12(opts, out, errOut, bs)
}

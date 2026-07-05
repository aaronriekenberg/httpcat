// Package client provides the HTTP client dispatcher for httpcat.
package client

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// bodySource represents a request body that may need to be read multiple times
// (for HTTP/3 fallback to HTTP/1.1 or HTTP/2).
// Exported as BodySource for testing.
type bodySource struct {
	spec   string    // The body spec from CLI (direct string, @-, or @filename)
	file   *os.File  // For files, kept open for seeking
	buffer []byte    // For stdin pipes that need retry (buffered eagerly)
	reader io.Reader // Current reader (may be reused)
	used   bool      // Whether this source has been read already
}

// newBodySource creates a bodySource for the given body spec.
// For stdin with potential HTTP/3 fallback, it buffers eagerly.
// For files, it opens them but defers reading.
func newBodySource(spec string) (*bodySource, error) {
	if spec == "" {
		return nil, nil
	}

	bs := &bodySource{spec: spec}

	// Handle file references (@filename or @-)
	if strings.HasPrefix(spec, "@") {
		path := spec[1:]
		if path == "-" {
			// stdin — check if seekable, else buffer it
			// We defer buffering decision to the caller (do function)
			// because we don't know if HTTP/3 is being used yet
			return bs, nil
		}
		// Open file for reading with seeking capability
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("opening body file %q: %w", path, err)
		}
		bs.file = file
		bs.reader = file
	}
	// For direct strings, we'll create a reader on-demand in getReader()

	return bs, nil
}

// bufferStdinIfNeeded buffers stdin for potential retry.
// This is called when HTTP/3 fallback is possible.
func (bs *bodySource) bufferStdinIfNeeded() error {
	if bs == nil || bs.spec == "" {
		return nil
	}

	spec := bs.spec
	if !strings.HasPrefix(spec, "@-") {
		// Not stdin
		return nil
	}

	if bs.buffer != nil {
		// Already buffered
		return nil
	}

	// Buffer stdin for retry capability
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin for buffering: %w", err)
	}
	bs.buffer = data
	return nil
}

// getReader returns an io.ReadCloser for this body source.
// If the source has been previously read, it seeks back to the start (if seekable)
// or returns the buffer (if stdin was buffered).
func (bs *bodySource) getReader() (io.ReadCloser, error) {
	if bs == nil {
		return nil, nil
	}

	spec := bs.spec
	if spec == "" {
		return nil, nil
	}

	// Handle direct strings (always fresh)
	if !strings.HasPrefix(spec, "@") {
		return io.NopCloser(strings.NewReader(spec)), nil
	}

	// Handle @filename or @-
	path := spec[1:]
	if path == "-" {
		// stdin
		if bs.buffer != nil {
			// Use buffered stdin for retry
			return io.NopCloser(bytes.NewReader(bs.buffer)), nil
		}
		if bs.used {
			// Unbuffered stdin already consumed; can't retry
			return nil, fmt.Errorf("cannot retry with stdin; stdin already consumed (use pipe buffering or file input)")
		}
		// First read: consume stdin directly (and mark as used for next attempt)
		bs.used = true
		return io.NopCloser(os.Stdin), nil
	}

	// Handle @filename: seek back to start
	if bs.file != nil {
		// Try to seek to start for retry
		if _, err := bs.file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("seeking in body file: %w", err)
		}
		return io.NopCloser(bs.file), nil
	}

	return nil, fmt.Errorf("body source not properly initialized")
}

// close closes the underlying file if open (but not stdin).
func (bs *bodySource) close() error {
	if bs != nil && bs.file != nil {
		return bs.file.Close()
	}
	return nil
}

// newRequest creates an *http.Request from options and applies custom headers.
// Uses a bodySource to handle potential retries (file seeking, etc).
func newRequest(opts *cli.Options, bs *bodySource) (*http.Request, error) {
	body, err := readBodyFromSource(opts, bs)
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
			if body != nil {
				body.Close()
			}
			return nil, err
		}
	}
	return req, nil
}

// readBodyFromSource reads the body from a bodySource.
// For direct strings and seekable files, this can be called multiple times.
func readBodyFromSource(opts *cli.Options, bs *bodySource) (io.ReadCloser, error) {
	bodySpec := opts.JSON
	if bodySpec == "" {
		bodySpec = opts.DataBinary
	}

	if bodySpec == "" {
		// No body
		return nil, nil
	}

	if bs == nil {
		// Shouldn't happen, but fall back to original logic
		return cli.ReadBody(opts)
	}

	return bs.getReader()
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
	// Create a bodySource that can be used for multiple request attempts
	// (for HTTP/3 fallback to HTTP/1.1 or HTTP/2).
	bodySpec := opts.JSON
	if bodySpec == "" {
		bodySpec = opts.DataBinary
	}

	bs, err := newBodySource(bodySpec)
	if err != nil {
		return fmt.Errorf("preparing request body: %w", err)
	}
	defer func() {
		if bs != nil {
			bs.close()
		}
	}()

	// If HTTP/3 is being attempted and we have stdin, buffer it for potential retry
	if opts.HTTP3 && !opts.HTTP3Only && strings.HasPrefix(bodySpec, "@-") {
		if err := bs.bufferStdinIfNeeded(); err != nil {
			return err
		}
	}

	if opts.HTTP3 {
		err := doHTTP3(opts, out, errOut, bs)
		if err == nil {
			return nil
		}
		if opts.HTTP3Only {
			return fmt.Errorf("HTTP/3 required but failed: %w", err)
		}
		// Fall back to HTTP/1.x or HTTP/2.
		return doHTTP12(opts, out, errOut, bs)
	}
	return doHTTP12(opts, out, errOut, bs)
}

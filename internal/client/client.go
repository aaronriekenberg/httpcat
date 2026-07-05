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
type bodySource struct {
	spec   string   // The body spec from CLI (direct string, @-, or @filename)
	file   *os.File // For files, kept open for seeking
	buffer []byte   // For stdin pipes that need retry (buffered eagerly)
	used   bool     // Whether stdin has been read without buffer
}

// newBodySource creates a bodySource for the given spec.
// Opens files but defers reading stdin.
func newBodySource(spec string) (*bodySource, error) {
	if spec == "" {
		return nil, nil
	}

	bs := &bodySource{spec: spec}

	// Handle file references (@filename or @-)
	if !strings.HasPrefix(spec, "@") {
		return bs, nil // Direct string
	}

	path := spec[1:]
	if path == "-" {
		return bs, nil // stdin — will be handled later
	}

	// Open file for reading with seeking
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening body file %q: %w", path, err)
	}
	bs.file = file
	return bs, nil
}

// bufferStdinIfNeeded buffers stdin for retry (called when HTTP/3 fallback possible).
func (bs *bodySource) bufferStdinIfNeeded() error {
	if bs == nil || bs.spec != "@-" {
		return nil
	}
	if bs.buffer != nil {
		return nil // Already buffered
	}

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	bs.buffer = data
	return nil
}

// getReader returns a fresh reader for this body source.
// For files, seeks to start. For buffered stdin, creates new reader from buffer.
func (bs *bodySource) getReader() (io.ReadCloser, error) {
	if bs == nil || bs.spec == "" {
		return nil, nil
	}

	// Direct string: always create fresh reader
	if !strings.HasPrefix(bs.spec, "@") {
		return io.NopCloser(strings.NewReader(bs.spec)), nil
	}

	path := bs.spec[1:]

	// stdin
	if path == "-" {
		if bs.buffer != nil {
			return io.NopCloser(bytes.NewReader(bs.buffer)), nil
		}
		if bs.used {
			return nil, fmt.Errorf("cannot retry with stdin; stdin already consumed")
		}
		bs.used = true
		return io.NopCloser(os.Stdin), nil
	}

	// file: seek to start
	if bs.file == nil {
		return nil, fmt.Errorf("body source not initialized")
	}
	if _, err := bs.file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("seeking in body file: %w", err)
	}
	return io.NopCloser(bs.file), nil
}

// close closes the underlying file if open.
func (bs *bodySource) close() error {
	if bs != nil && bs.file != nil {
		return bs.file.Close()
	}
	return nil
}

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

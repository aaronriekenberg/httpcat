// Package client provides HTTP/3 request execution using quic-go.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"os"

	"github.com/quic-go/quic-go/http3"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/verbose"
)

// DoHTTP3 executes an HTTP/3 request using quic-go.
// Response body is written to stdout; verbose info goes to stderr.
func DoHTTP3(opts *cli.Options) error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: opts.Insecure, //nolint:gosec // intentional per -k flag
	}

	rt := &http3.Transport{
		TLSClientConfig: tlsCfg,
	}
	defer rt.Close()

	if opts.Verbose {
		verbose.PrintInfo(os.Stderr, "Using HTTP/3 (QUIC)")
	}

	req, err := newRequest(opts)
	if err != nil {
		return err
	}

	if opts.Verbose {
		verbose.PrintInfo(os.Stderr, "Connecting to %s", req.Host)
		verbose.PrintRequest(os.Stderr, req)
	}

	resp, err := rt.RoundTrip(req)
	if err != nil {
		return fmt.Errorf("HTTP/3 request failed: %w", err)
	}
	defer resp.Body.Close()

	if opts.Verbose {
		verbose.PrintInfo(os.Stderr, "Protocol: %s", resp.Proto)
		verbose.PrintResponse(os.Stderr, resp)
	}

	if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	return nil
}

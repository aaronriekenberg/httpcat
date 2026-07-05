// Package client provides HTTP/3 request execution using quic-go.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/quic-go/quic-go/http3"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/verbose"
)

// DoHTTP3 executes an HTTP/3 request using quic-go.
// Response body is written to stdout; verbose info goes to stderr.
func DoHTTP3(opts *cli.Options) error {
	return doHTTP3(opts, os.Stdout, os.Stderr, nil)
}

// doHTTP3 is the testable implementation that writes body to out and
// verbose/error info to errOut. bs is a bodySource for potential retries.
func doHTTP3(opts *cli.Options, out, errOut io.Writer, bs *bodySource) error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: opts.Insecure, //nolint:gosec // intentional per -k flag
	}

	rt := &http3.Transport{
		TLSClientConfig: tlsCfg,
	}
	defer rt.Close()

	client := &http.Client{Transport: rt}

	if opts.Verbose {
		verbose.PrintInfo(errOut, "Using HTTP/3 (QUIC)")
	}

	req, err := newRequest(opts, bs)
	if err != nil {
		return err
	}

	if opts.Verbose {
		verbose.PrintInfo(errOut, "Connecting to %s", req.Host)
		verbose.PrintRequest(errOut, req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP/3 request failed: %w", err)
	}
	defer resp.Body.Close()

	if opts.Verbose {
		verbose.PrintInfo(errOut, "Protocol: %s", resp.Proto)
		verbose.PrintResponse(errOut, resp)
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	return nil
}

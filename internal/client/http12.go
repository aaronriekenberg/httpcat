// Package client provides HTTP/1.1 and HTTP/2 request execution using net/http.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/verbose"
)

// DoHTTP12 executes an HTTP/1.1 or HTTP/2 request according to opts.
// Response body is written to stdout; verbose info goes to stderr.
func DoHTTP12(opts *cli.Options) error {
	return doHTTP12(opts, os.Stdout, os.Stderr, nil)
}

// doHTTP12 is the testable implementation that writes body to out and
// verbose/error info to errOut. bs is a bodySource for potential retries.
func doHTTP12(opts *cli.Options, out, errOut io.Writer, bs *bodySource) error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: opts.Insecure, //nolint:gosec // intentional per -k flag
	}

	transport := &http.Transport{
		TLSClientConfig: tlsCfg,
	}

	if opts.HTTP2PriorKnowledge {
		// Set protocol based on scheme:
		//   http://  → UnencryptedHTTP2 (h2c, plain TCP)
		//   https:// → HTTP2 (TLS + ALPN)
		var p http.Protocols
		if strings.HasPrefix(opts.URL, "https://") {
			if opts.Verbose {
				verbose.PrintInfo(errOut, "Using HTTP/2 (TLS + ALPN)")
			}
			p.SetHTTP2(true)
		} else {
			if opts.Verbose {
				verbose.PrintInfo(errOut, "Using H2C (HTTP/2 over plain TCP)")
			}
			p.SetUnencryptedHTTP2(true)
		}
		transport.Protocols = &p
	} else {
		if opts.Verbose {
			verbose.PrintInfo(errOut, "Using HTTP/1.1 or HTTP/2 (negotiated via ALPN)")
		}
		// Default: negotiate HTTP/1.1 or HTTP/2 via ALPN.
		var p http.Protocols
		p.SetHTTP1(true)
		p.SetHTTP2(true)
		transport.Protocols = &p
	}

	client := &http.Client{Transport: transport}

	req, err := newRequest(opts, bs)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	if opts.Verbose {
		verbose.PrintInfo(errOut, "Connecting to %s", req.Host)
		verbose.PrintRequest(errOut, req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
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

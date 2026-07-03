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
	return doHTTP12(opts, os.Stdout, os.Stderr)
}

// doHTTP12 is the testable implementation that writes body to out and
// verbose/error info to errOut.
func doHTTP12(opts *cli.Options, out, errOut io.Writer) error {
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
			p.SetHTTP2(true)
		} else {
			p.SetUnencryptedHTTP2(true)
		}
		transport.Protocols = &p
		if opts.Verbose {
			verbose.PrintInfo(errOut, "Using HTTP/2 with prior knowledge")
		}
	} else {
		// Default: negotiate HTTP/1.1 or HTTP/2 via ALPN.
		var p http.Protocols
		p.SetHTTP1(true)
		p.SetHTTP2(true)
		transport.Protocols = &p
	}

	client := &http.Client{Transport: transport}

	req, err := newRequest(opts)
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

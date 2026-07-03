// Package client provides HTTP/1.1 and HTTP/2 request execution using net/http.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/net/http2"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/verbose"
)

// DoHTTP12 executes an HTTP/1.1 or HTTP/2 request according to opts.
// Response body is written to stdout; verbose info goes to stderr.
func DoHTTP12(opts *cli.Options) error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: opts.Insecure, //nolint:gosec // intentional per -k flag
	}

	var transport http.RoundTripper

	if opts.HTTP2PriorKnowledge {
		// Force HTTP/2 without TLS upgrade negotiation (h2c or direct h2).
		t2 := &http2.Transport{
			AllowHTTP:       true, // permit plain-text h2c
			TLSClientConfig: tlsCfg,
			DialTLSContext:  nil, // use default
		}
		transport = t2
		if opts.Verbose {
			verbose.PrintInfo(os.Stderr, "Using HTTP/2 with prior knowledge")
		}
	} else {
		transport = &http.Transport{
			TLSClientConfig:   tlsCfg,
			ForceAttemptHTTP2: true, // negotiate HTTP/2 via ALPN when HTTPS
		}
	}

	client := &http.Client{Transport: transport}

	req, err := http.NewRequest(opts.Method, opts.URL, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}

	if opts.Verbose {
		// Proto isn't known until after the round-trip; print what we know.
		verbose.PrintInfo(os.Stderr, "Connecting to %s", req.Host)
		verbose.PrintRequest(os.Stderr, req)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
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

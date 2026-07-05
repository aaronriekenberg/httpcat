// Package client provides HTTP/1.1 and HTTP/2 request execution using net/http.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/verbose"
)

// doHTTP12 executes an HTTP/1.1 or HTTP/2 request according to opts.
func doHTTP12(opts *cli.Options, out, errOut io.Writer, bs *bodySource) error {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.Insecure, //nolint:gosec
		},
	}

	// Configure protocol preferences
	var p http.Protocols
	if opts.HTTP2PriorKnowledge {
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
	} else {
		if opts.Verbose {
			verbose.PrintInfo(errOut, "Using HTTP/1.1 or HTTP/2 (negotiated via ALPN)")
		}
		p.SetHTTP1(true)
		p.SetHTTP2(true)
	}
	transport.Protocols = &p

	req, err := newRequest(opts, bs)
	if err != nil {
		return err
	}

	if opts.Verbose {
		verbose.PrintInfo(errOut, "Connecting to %s", req.Host)
		verbose.PrintRequest(errOut, req)
	}

	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if opts.Verbose {
		verbose.PrintInfo(errOut, "Protocol: %s", resp.Proto)
		verbose.PrintResponse(errOut, resp)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}

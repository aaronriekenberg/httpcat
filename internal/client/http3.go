// Package client provides HTTP/3 request execution using quic-go.
package client

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"

	"github.com/quic-go/quic-go/http3"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/verbose"
)

// doHTTP3 executes an HTTP/3 request using quic-go.
func doHTTP3(opts *cli.Options, out, errOut io.Writer, bs *bodySource) error {
	client := &http.Client{
		Transport: &http3.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.Insecure, //nolint:gosec
			},
		},
	}
	defer client.Transport.(*http3.Transport).Close()

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

	_, err = io.Copy(out, resp.Body)
	return err
}

// export_test.go exposes internal functions for use by the client_test package.
// This file is only compiled during tests.
package client

import (
	"io"
	"net/http"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// DoWithWriters is a test hook that calls the internal do() with explicit
// writers so tests can capture stdout/stderr without redirecting os.Stdout.
func DoWithWriters(opts *cli.Options, out, errOut io.Writer) error {
	return do(opts, out, errOut)
}

// ApplyHeader is a test hook that exposes the internal applyHeader function.
func ApplyHeader(req *http.Request, header string) error {
	return applyHeader(req, header)
}

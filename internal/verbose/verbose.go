// Package verbose provides curl-style verbose output for request/response headers.
package verbose

import (
	"fmt"
	"io"
	"net/http"
	"sort"
)

// PrintRequest writes request metadata to w in curl verbose format.
// Lines are prefixed with "> ".
func PrintRequest(w io.Writer, req *http.Request) {
	fmt.Fprintf(w, "> %s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto)
	fmt.Fprintf(w, "> Host: %s\r\n", req.Host)

	keys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range req.Header[k] {
			fmt.Fprintf(w, "> %s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(w, ">\r\n")
}

// PrintResponse writes response metadata to w in curl verbose format.
// Lines are prefixed with "< ".
func PrintResponse(w io.Writer, resp *http.Response) {
	fmt.Fprintf(w, "< %s\r\n", resp.Status)

	keys := make([]string, 0, len(resp.Header))
	for k := range resp.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range resp.Header[k] {
			fmt.Fprintf(w, "< %s: %s\r\n", k, v)
		}
	}
	fmt.Fprintf(w, "<\r\n")
}

// PrintInfo writes an informational line to w.
// Lines are prefixed with "* ".
func PrintInfo(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "* "+format+"\n", args...)
}

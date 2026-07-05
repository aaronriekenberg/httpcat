// Package verbose provides curl-style verbose output for request/response headers.
package verbose

import (
	"fmt"
	"io"
	"net/http"
	"sort"
)

// printHeaders writes sorted headers to w with the given prefix.
func printHeaders(w io.Writer, headers http.Header, prefix string) {
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		for _, v := range headers[k] {
			fmt.Fprintf(w, "%s %s: %s\r\n", prefix, k, v)
		}
	}
}

// PrintRequest writes request metadata to w in curl verbose format.
func PrintRequest(w io.Writer, req *http.Request) {
	fmt.Fprintf(w, "> %s %s\r\n", req.Method, req.URL.RequestURI())
	fmt.Fprintf(w, "> Host: %s\r\n", req.Host)
	printHeaders(w, req.Header, ">")
	fmt.Fprintf(w, ">\r\n")
}

// PrintResponse writes response metadata to w in curl verbose format.
func PrintResponse(w io.Writer, resp *http.Response) {
	fmt.Fprintf(w, "< %s\r\n", resp.Status)
	printHeaders(w, resp.Header, "<")
	fmt.Fprintf(w, "<\r\n")
}

// PrintInfo writes an informational line to w with "* " prefix.
func PrintInfo(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, "* "+format+"\n", args...)
}

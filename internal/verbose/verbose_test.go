package verbose_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/aaronriekenberg/httpcat/internal/verbose"
)

// ---------------------------------------------------------------------------
// PrintInfo
// ---------------------------------------------------------------------------

func TestPrintInfo(t *testing.T) {
	var b strings.Builder
	verbose.PrintInfo(&b, "hello %s", "world")
	got := b.String()
	if got != "* hello world\n" {
		t.Errorf("PrintInfo output = %q, want %q", got, "* hello world\n")
	}
}

func TestPrintInfoNoArgs(t *testing.T) {
	var b strings.Builder
	verbose.PrintInfo(&b, "plain message")
	if !strings.HasPrefix(b.String(), "* ") {
		t.Errorf("PrintInfo should start with '* ', got %q", b.String())
	}
}

// ---------------------------------------------------------------------------
// PrintRequest
// ---------------------------------------------------------------------------

func TestPrintRequestFirstLine(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/path?q=1", nil)
	var b strings.Builder
	verbose.PrintRequest(&b, req)

	lines := strings.Split(b.String(), "\r\n")
	if len(lines) < 1 {
		t.Fatal("no output lines")
	}
	want := "> GET /path?q=1 HTTP/1.1"
	if lines[0] != want {
		t.Errorf("first line = %q, want %q", lines[0], want)
	}
}

func TestPrintRequestHostLine(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/", nil)
	var b strings.Builder
	verbose.PrintRequest(&b, req)

	if !strings.Contains(b.String(), "> Host: example.com\r\n") {
		t.Errorf("output missing host line, got:\n%s", b.String())
	}
}

func TestPrintRequestHeaders(t *testing.T) {
	req, _ := http.NewRequest("POST", "https://example.com/", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Custom", "value")

	var b strings.Builder
	verbose.PrintRequest(&b, req)
	out := b.String()

	if !strings.Contains(out, "> Content-Type: application/json\r\n") {
		t.Errorf("missing Content-Type header in:\n%s", out)
	}
	if !strings.Contains(out, "> X-Custom: value\r\n") {
		t.Errorf("missing X-Custom header in:\n%s", out)
	}
}

func TestPrintRequestHeadersAreSorted(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/", nil)
	req.Header.Set("Z-Last", "z")
	req.Header.Set("A-First", "a")

	var b strings.Builder
	verbose.PrintRequest(&b, req)
	out := b.String()

	posA := strings.Index(out, "A-First")
	posZ := strings.Index(out, "Z-Last")
	if posA < 0 || posZ < 0 {
		t.Fatalf("headers not found in output:\n%s", out)
	}
	if posA > posZ {
		t.Errorf("headers not sorted: A-First at %d, Z-Last at %d", posA, posZ)
	}
}

func TestPrintRequestEndsWithBlankLine(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/", nil)
	var b strings.Builder
	verbose.PrintRequest(&b, req)

	if !strings.HasSuffix(b.String(), ">\r\n") {
		t.Errorf("output should end with blank '>' line, got:\n%s", b.String())
	}
}

func TestPrintRequestLinePrefix(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://example.com/", nil)
	var b strings.Builder
	verbose.PrintRequest(&b, req)

	for _, line := range strings.Split(strings.TrimRight(b.String(), "\r\n"), "\r\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "> ") && line != ">" {
			t.Errorf("line missing '> ' prefix: %q", line)
		}
	}
}

// ---------------------------------------------------------------------------
// PrintResponse
// ---------------------------------------------------------------------------

func TestPrintResponseStatusLine(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{},
	}
	var b strings.Builder
	verbose.PrintResponse(&b, resp)

	lines := strings.Split(b.String(), "\r\n")
	want := "< 200 OK"
	if lines[0] != want {
		t.Errorf("first line = %q, want %q", lines[0], want)
	}
}

func TestPrintResponseHeaders(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header: http.Header{
			"Content-Type": {"application/json"},
			"X-Request-Id": {"abc123"},
		},
	}
	var b strings.Builder
	verbose.PrintResponse(&b, resp)
	out := b.String()

	if !strings.Contains(out, "< Content-Type: application/json\r\n") {
		t.Errorf("missing Content-Type in:\n%s", out)
	}
	if !strings.Contains(out, "< X-Request-Id: abc123\r\n") {
		t.Errorf("missing X-Request-Id in:\n%s", out)
	}
}

func TestPrintResponseHeadersAreSorted(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header: http.Header{
			"Z-Header": {"z"},
			"A-Header": {"a"},
		},
	}
	var b strings.Builder
	verbose.PrintResponse(&b, resp)
	out := b.String()

	posA := strings.Index(out, "A-Header")
	posZ := strings.Index(out, "Z-Header")
	if posA < 0 || posZ < 0 {
		t.Fatalf("headers not found:\n%s", out)
	}
	if posA > posZ {
		t.Errorf("headers not sorted: A-Header at %d, Z-Header at %d", posA, posZ)
	}
}

func TestPrintResponseEndsWithBlankLine(t *testing.T) {
	resp := &http.Response{
		Status:     "404 Not Found",
		StatusCode: 404,
		Header:     http.Header{},
	}
	var b strings.Builder
	verbose.PrintResponse(&b, resp)

	if !strings.HasSuffix(b.String(), "<\r\n") {
		t.Errorf("output should end with blank '<' line, got:\n%s", b.String())
	}
}

func TestPrintResponseLinePrefix(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header:     http.Header{"Content-Length": {"0"}},
	}
	var b strings.Builder
	verbose.PrintResponse(&b, resp)

	for _, line := range strings.Split(strings.TrimRight(b.String(), "\r\n"), "\r\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "< ") && line != "<" {
			t.Errorf("line missing '< ' prefix: %q", line)
		}
	}
}

func TestPrintResponseMultiValueHeader(t *testing.T) {
	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: 200,
		Header: http.Header{
			"Set-Cookie": {"a=1", "b=2"},
		},
	}
	var b strings.Builder
	verbose.PrintResponse(&b, resp)
	out := b.String()

	if strings.Count(out, "Set-Cookie") != 2 {
		t.Errorf("expected 2 Set-Cookie lines, got:\n%s", out)
	}
}

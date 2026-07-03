package cli_test

import (
	"testing"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// parseOK is a helper that asserts Parse succeeds and returns the Options.
func parseOK(t *testing.T, args ...string) *cli.Options {
	t.Helper()
	opts, err := cli.Parse(args)
	if err != nil {
		t.Fatalf("Parse(%v) unexpected error: %v", args, err)
	}
	return opts
}

// parseErr is a helper that asserts Parse returns an error containing substr.
func parseErr(t *testing.T, substr string, args ...string) {
	t.Helper()
	_, err := cli.Parse(args)
	if err == nil {
		t.Fatalf("Parse(%v) expected error containing %q, got nil", args, substr)
	}
	if substr != "" && !contains(err.Error(), substr) {
		t.Fatalf("Parse(%v) error %q does not contain %q", args, err.Error(), substr)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || len(s) >= len(sub) && (s == sub || len(s) > 0 && containsRune(s, sub))
}

func containsRune(s, sub string) bool {
	for i := range s {
		if i+len(sub) <= len(s) && s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Defaults
// ---------------------------------------------------------------------------

func TestDefaultMethod(t *testing.T) {
	opts := parseOK(t, "https://example.com")
	if opts.Method != "GET" {
		t.Errorf("expected Method=GET, got %q", opts.Method)
	}
}

func TestDefaultFlags(t *testing.T) {
	opts := parseOK(t, "https://example.com")
	if opts.Insecure {
		t.Error("Insecure should default to false")
	}
	if opts.Verbose {
		t.Error("Verbose should default to false")
	}
	if opts.HTTP3 {
		t.Error("HTTP3 should default to false")
	}
	if opts.HTTP3Only {
		t.Error("HTTP3Only should default to false")
	}
	if opts.HTTP2PriorKnowledge {
		t.Error("HTTP2PriorKnowledge should default to false")
	}
}

// ---------------------------------------------------------------------------
// URL handling
// ---------------------------------------------------------------------------

func TestURLCapture(t *testing.T) {
	opts := parseOK(t, "https://example.com/path?q=1")
	if opts.URL != "https://example.com/path?q=1" {
		t.Errorf("unexpected URL %q", opts.URL)
	}
}

func TestHTTPScheme(t *testing.T) {
	opts := parseOK(t, "http://example.com")
	if opts.URL != "http://example.com" {
		t.Errorf("unexpected URL %q", opts.URL)
	}
}

func TestNoURL(t *testing.T) {
	parseErr(t, "no URL", "-v")
}

func TestMultipleURLs(t *testing.T) {
	parseErr(t, "multiple URLs", "https://a.com", "https://b.com")
}

func TestUnsupportedScheme(t *testing.T) {
	parseErr(t, "unsupported scheme", "ftp://example.com")
	parseErr(t, "unsupported scheme", "ws://example.com")
}

// ---------------------------------------------------------------------------
// -X / --request
// ---------------------------------------------------------------------------

func TestShortMethod(t *testing.T) {
	opts := parseOK(t, "-X", "POST", "https://example.com")
	if opts.Method != "POST" {
		t.Errorf("expected POST, got %q", opts.Method)
	}
}

func TestLongMethod(t *testing.T) {
	opts := parseOK(t, "--request", "DELETE", "https://example.com")
	if opts.Method != "DELETE" {
		t.Errorf("expected DELETE, got %q", opts.Method)
	}
}

func TestMethodUpcased(t *testing.T) {
	opts := parseOK(t, "-X", "put", "https://example.com")
	if opts.Method != "PUT" {
		t.Errorf("expected PUT, got %q", opts.Method)
	}
}

func TestMethodEqualSyntax(t *testing.T) {
	opts := parseOK(t, "--request=PATCH", "https://example.com")
	if opts.Method != "PATCH" {
		t.Errorf("expected PATCH, got %q", opts.Method)
	}
}

func TestMethodInlineShort(t *testing.T) {
	// -XPOST (value glued to short flag)
	opts := parseOK(t, "-XPOST", "https://example.com")
	if opts.Method != "POST" {
		t.Errorf("expected POST, got %q", opts.Method)
	}
}

func TestMethodMissingArg(t *testing.T) {
	parseErr(t, "requires an argument", "-X")
	parseErr(t, "requires an argument", "--request")
}

// ---------------------------------------------------------------------------
// -k / --insecure
// ---------------------------------------------------------------------------

func TestInsecureShort(t *testing.T) {
	opts := parseOK(t, "-k", "https://example.com")
	if !opts.Insecure {
		t.Error("expected Insecure=true")
	}
}

func TestInsecureLong(t *testing.T) {
	opts := parseOK(t, "--insecure", "https://example.com")
	if !opts.Insecure {
		t.Error("expected Insecure=true")
	}
}

// ---------------------------------------------------------------------------
// -v / --verbose
// ---------------------------------------------------------------------------

func TestVerboseShort(t *testing.T) {
	opts := parseOK(t, "-v", "https://example.com")
	if !opts.Verbose {
		t.Error("expected Verbose=true")
	}
}

func TestVerboseLong(t *testing.T) {
	opts := parseOK(t, "--verbose", "https://example.com")
	if !opts.Verbose {
		t.Error("expected Verbose=true")
	}
}

// ---------------------------------------------------------------------------
// --http2-prior-knowledge
// ---------------------------------------------------------------------------

func TestHTTP2PriorKnowledge(t *testing.T) {
	opts := parseOK(t, "--http2-prior-knowledge", "https://example.com")
	if !opts.HTTP2PriorKnowledge {
		t.Error("expected HTTP2PriorKnowledge=true")
	}
}

// ---------------------------------------------------------------------------
// --http3 / --http3-only
// ---------------------------------------------------------------------------

func TestHTTP3(t *testing.T) {
	opts := parseOK(t, "--http3", "https://example.com")
	if !opts.HTTP3 {
		t.Error("expected HTTP3=true")
	}
	if opts.HTTP3Only {
		t.Error("HTTP3Only should be false")
	}
}

func TestHTTP3Only(t *testing.T) {
	opts := parseOK(t, "--http3-only", "https://example.com")
	if !opts.HTTP3Only {
		t.Error("expected HTTP3Only=true")
	}
	// --http3-only implies --http3
	if !opts.HTTP3 {
		t.Error("HTTP3Only should imply HTTP3=true")
	}
}

// ---------------------------------------------------------------------------
// Mutual exclusion
// ---------------------------------------------------------------------------

func TestHTTP3AndPriorKnowledgeMutuallyExclusive(t *testing.T) {
	parseErr(t, "mutually exclusive", "--http3", "--http2-prior-knowledge", "https://example.com")
}

func TestHTTP3OnlyAndPriorKnowledgeMutuallyExclusive(t *testing.T) {
	parseErr(t, "mutually exclusive", "--http3-only", "--http2-prior-knowledge", "https://example.com")
}

// ---------------------------------------------------------------------------
// Bundled short flags
// ---------------------------------------------------------------------------

func TestBundledShortFlags(t *testing.T) {
	opts := parseOK(t, "-kv", "https://example.com")
	if !opts.Insecure {
		t.Error("expected Insecure=true")
	}
	if !opts.Verbose {
		t.Error("expected Verbose=true")
	}
}

func TestBundledShortFlagsWithMethod(t *testing.T) {
	// -kvXPOST
	opts := parseOK(t, "-kvXPOST", "https://example.com")
	if !opts.Insecure {
		t.Error("expected Insecure=true")
	}
	if !opts.Verbose {
		t.Error("expected Verbose=true")
	}
	if opts.Method != "POST" {
		t.Errorf("expected POST, got %q", opts.Method)
	}
}

// ---------------------------------------------------------------------------
// Unknown flags
// ---------------------------------------------------------------------------

func TestUnknownLongFlag(t *testing.T) {
	parseErr(t, "unknown option", "--no-such-flag", "https://example.com")
}

func TestUnknownShortFlag(t *testing.T) {
	parseErr(t, "unknown option", "-z", "https://example.com")
}

// ---------------------------------------------------------------------------
// Flag/URL ordering
// ---------------------------------------------------------------------------

func TestFlagsAfterURL(t *testing.T) {
	opts := parseOK(t, "https://example.com", "-v")
	if !opts.Verbose {
		t.Error("expected Verbose=true when flag appears after URL")
	}
}

func TestFlagsBeforeAndAfterURL(t *testing.T) {
	opts := parseOK(t, "-k", "https://example.com", "-v")
	if !opts.Insecure {
		t.Error("expected Insecure=true")
	}
	if !opts.Verbose {
		t.Error("expected Verbose=true")
	}
}

// ---------------------------------------------------------------------------
// --version / -V
// ---------------------------------------------------------------------------

func TestVersionLong(t *testing.T) {
	opts := parseOK(t, "--version")
	if !opts.Version {
		t.Error("expected Version=true")
	}
}

func TestVersionShort(t *testing.T) {
	opts := parseOK(t, "-V")
	if !opts.Version {
		t.Error("expected Version=true")
	}
}

// --version needs no URL.
func TestVersionNoURLRequired(t *testing.T) {
	opts, err := cli.Parse([]string{"--version"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !opts.Version {
		t.Error("expected Version=true")
	}
	if opts.URL != "" {
		t.Errorf("expected empty URL, got %q", opts.URL)
	}
}

// --version alongside a URL is also fine.
func TestVersionWithURL(t *testing.T) {
	opts := parseOK(t, "--version", "https://example.com")
	if !opts.Version {
		t.Error("expected Version=true")
	}
}

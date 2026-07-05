package cli_test

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/aaronriekenberg/httpcat/internal/cli"
)

// readBody reads the request body from options.
// Supports:
//   - Direct string: "hello world"
//   - stdin: "@-"
//   - file: "@/path/to/file"
//
// Returns io.ReadCloser or error if reading fails.
func readBody(opts *cli.Options) (io.ReadCloser, error) {
	bodySpec := opts.JSON
	if bodySpec == "" {
		bodySpec = opts.DataBinary
	}

	if bodySpec == "" {
		// No body
		return nil, nil
	}

	// Check for special prefixes
	if strings.HasPrefix(bodySpec, "@-") {
		// Read from stdin
		return os.Stdin, nil
	}

	if strings.HasPrefix(bodySpec, "@") {
		// Read from file
		filePath := bodySpec[1:]
		file, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("opening body file %q: %w", filePath, err)
		}
		return file, nil
	}

	// Direct string
	return io.NopCloser(strings.NewReader(bodySpec)), nil
}

// ReadBody tests
// ---------------------------------------------------------------------------

func TestReadBodyNil(t *testing.T) {
	opts := &cli.Options{}
	body, err := readBody(opts)
	if err != nil {
		t.Fatalf("readBody() unexpected error: %v", err)
	}
	if body != nil {
		t.Errorf("expected body=nil, got %v", body)
	}
}

func TestReadBodyDirectString(t *testing.T) {
	opts := &cli.Options{
		JSON: `{"test": "data"}`,
	}
	body, err := readBody(opts)
	if err != nil {
		t.Fatalf("readBody() unexpected error: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() unexpected error: %v", err)
	}
	if string(data) != `{"test": "data"}` {
		t.Errorf("expected %q, got %q", `{"test": "data"}`, string(data))
	}
}

func TestReadBodyFromFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "httpcat-test-*.json")
	if err != nil {
		t.Fatalf("CreateTemp() unexpected error: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testContent := `{"file": "content"}`
	if _, err := tmpFile.WriteString(testContent); err != nil {
		t.Fatalf("WriteString() unexpected error: %v", err)
	}
	tmpFile.Close()

	opts := &cli.Options{
		JSON: "@" + tmpFile.Name(),
	}
	body, err := readBody(opts)
	if err != nil {
		t.Fatalf("readBody() unexpected error: %v", err)
	}
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() unexpected error: %v", err)
	}
	if string(data) != testContent {
		t.Errorf("expected %q, got %q", testContent, string(data))
	}
}

func TestReadBodyFileNotFound(t *testing.T) {
	opts := &cli.Options{
		JSON: "@/nonexistent/path/file.json",
	}
	_, err := readBody(opts)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
	if !contains(err.Error(), "opening body file") {
		t.Errorf("expected error message to contain 'opening body file', got %q", err.Error())
	}
}

func TestReadBodyDataBinaryPreference(t *testing.T) {
	// When both are empty, body should be nil
	opts := &cli.Options{}
	body, err := readBody(opts)
	if err != nil {
		t.Fatalf("readBody() unexpected error: %v", err)
	}
	if body != nil {
		t.Errorf("expected nil body when both are empty")
	}
}

func TestReadBodyWithBinaryContent(t *testing.T) {
	// Create a temporary file with binary content
	tmpFile, err := os.CreateTemp("", "httpcat-test-*.bin")
	if err != nil {
		t.Fatalf("CreateTemp() unexpected error: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	binaryContent := []byte{0x00, 0x01, 0x02, 0xFF}
	if _, err := tmpFile.Write(binaryContent); err != nil {
		t.Fatalf("Write() unexpected error: %v", err)
	}
	tmpFile.Close()

	opts := &cli.Options{
		DataBinary: "@" + tmpFile.Name(),
	}
	body, err := readBody(opts)
	if err != nil {
		t.Fatalf("readBody() unexpected error: %v", err)
	}
	defer body.Close()

	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll() unexpected error: %v", err)
	}
	if len(data) != len(binaryContent) {
		t.Errorf("expected length %d, got %d", len(binaryContent), len(data))
	}
	for i, b := range data {
		if b != binaryContent[i] {
			t.Errorf("byte %d: expected %02x, got %02x", i, binaryContent[i], b)
		}
	}
}

package client_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aaronriekenberg/httpcat/internal/cli"
	"github.com/aaronriekenberg/httpcat/internal/client"
)

// ---------------------------------------------------------------------------
// bodySource Tests
// ---------------------------------------------------------------------------

// TestNewBodySourceDirectString tests that direct strings work
func TestNewBodySourceDirectString(t *testing.T) {
	bs, err := client.NewBodySource("hello world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bs == nil {
		t.Fatal("expected non-nil bodySource")
	}

	// Get reader and verify content
	reader, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error getting reader: %v", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("unexpected error reading: %v", err)
	}
	if string(data) != "hello world" {
		t.Errorf("got %q, want %q", string(data), "hello world")
	}

	// Clean up
	if err := bs.Close(); err != nil {
		t.Errorf("unexpected error closing: %v", err)
	}
}

// TestNewBodySourceDirectStringRetry tests multiple reads of same string
func TestNewBodySourceDirectStringRetry(t *testing.T) {
	bs, err := client.NewBodySource("test content")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// First read
	reader1, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error on first read: %v", err)
	}
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	// Second read (should work fine for strings)
	reader2, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error on second read: %v", err)
	}
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != string(data2) {
		t.Errorf("reads differ: %q vs %q", string(data1), string(data2))
	}

	bs.Close()
}

// TestNewBodySourceEmptyString tests empty body spec
func TestNewBodySourceEmptyString(t *testing.T) {
	bs, err := client.NewBodySource("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bs != nil {
		t.Fatal("expected nil bodySource for empty spec")
	}
}

// TestNewBodySourceFile tests file-based body with seeking
func TestNewBodySourceFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "httpcat-test-*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testData := "file content for testing"
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	tmpFile.Close()

	// Create bodySource for the file
	spec := "@" + tmpFile.Name()
	bs, err := client.NewBodySource(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer bs.Close()

	// First read
	reader1, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error on first read: %v", err)
	}
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	// Second read (should seek back and work)
	reader2, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error on second read: %v", err)
	}
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != testData || string(data2) != testData {
		t.Errorf("file content mismatch: got %q and %q, want %q", string(data1), string(data2), testData)
	}
}

// TestNewBodySourceFileNotFound tests handling of missing files
func TestNewBodySourceFileNotFound(t *testing.T) {
	spec := "@/nonexistent/file/path"
	bs, err := client.NewBodySource(spec)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if bs != nil {
		t.Error("expected nil bodySource on error")
	}
	if !strings.Contains(err.Error(), "opening body file") {
		t.Errorf("error should mention file opening: %v", err)
	}
}

// TestNewBodySourceLargeFile tests that large files don't buffer to memory
func TestNewBodySourceLargeFile(t *testing.T) {
	// Create a temporary file with 5MB of data
	tmpFile, err := os.CreateTemp("", "httpcat-large-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 5MB of data
	chunkSize := 1024 * 1024 // 1MB
	chunk := make([]byte, chunkSize)
	for i := 0; i < 5; i++ {
		if _, err := tmpFile.Write(chunk); err != nil {
			t.Fatalf("failed to write data: %v", err)
		}
	}
	tmpFile.Close()

	// Create bodySource for large file
	spec := "@" + tmpFile.Name()
	bs, err := client.NewBodySource(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer bs.Close()

	// Verify we can seek and read
	reader, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer reader.Close()

	// Read some data and verify
	data := make([]byte, 1024)
	n, err := reader.Read(data)
	if n == 0 || err != nil {
		t.Fatalf("failed to read from large file: %v", err)
	}
}

// TestBufferStdinIfNeeded tests stdin buffering for retry
func TestBufferStdinIfNeeded(t *testing.T) {
	// Create a pipe with test data
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	// Write test data and close write end
	testData := "stdin test data"
	go func() {
		w.WriteString(testData)
		w.Close()
	}()

	// Temporarily replace os.Stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Create bodySource for stdin
	bs, err := client.NewBodySource("@-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer bs.Close()

	// Buffer stdin
	if err := bs.BufferStdinIfNeeded(); err != nil {
		t.Fatalf("unexpected error buffering stdin: %v", err)
	}

	// First read (from buffer)
	reader1, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error on first read: %v", err)
	}
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	// Second read (should work because stdin was buffered)
	reader2, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error on second read: %v", err)
	}
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != testData || string(data2) != testData {
		t.Errorf("stdin content mismatch: got %q and %q, want %q", string(data1), string(data2), testData)
	}
}

// TestStdinRetryWithoutBuffer tests that retry without buffer fails
func TestStdinRetryWithoutBuffer(t *testing.T) {
	// Create a pipe with test data
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	// Write small data and close
	go func() {
		w.WriteString("data")
		w.Close()
	}()

	// Replace stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Create bodySource for stdin
	bs, err := client.NewBodySource("@-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer bs.Close()

	// First read (should work)
	reader1, err := bs.GetReader()
	if err != nil {
		t.Fatalf("unexpected error on first read: %v", err)
	}
	io.ReadAll(reader1)
	reader1.Close()

	// Second read without buffering (should fail)
	_, err = bs.GetReader()
	if err == nil {
		t.Error("expected error on retry without buffer, got nil")
	}
	if !strings.Contains(err.Error(), "cannot retry with stdin") {
		t.Errorf("error should mention stdin retry: %v", err)
	}
}

// TestHTTP3FallbackWithStdin tests stdin with HTTP/3 fallback
func TestHTTP3FallbackWithStdin(t *testing.T) {
	// Create a pipe with JSON data
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	testJSON := `{"test":"http3fallback"}`
	go func() {
		w.WriteString(testJSON)
		w.Close()
	}()

	// Replace stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Create a test server
	srv := startServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	// Create options for HTTP/3 with stdin
	opts := &cli.Options{
		Method:  "POST",
		URL:     srv.URL,
		JSON:    "@-",
		HTTP3:   true,
		Headers: []string{},
	}

	// Execute request (will fallback from HTTP/3 to HTTP/1.1)
	var out, errOut bytes.Buffer
	err = client.DoWithWriters(opts, &out, &errOut)
	if err != nil {
		// Expected to succeed (fallback should work)
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response contains our data
	if !strings.Contains(out.String(), "test") {
		t.Errorf("response should contain test data, got: %s", out.String())
	}
}

// TestHTTP3FallbackWithFile tests file with HTTP/3 fallback
func TestHTTP3FallbackWithFile(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "httpcat-test-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	testJSON := `{"source":"file","size":1024}`
	if _, err := tmpFile.WriteString(testJSON); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	tmpFile.Close()

	// Create a test server
	srv := startServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	// Create options for HTTP/3 with file
	opts := &cli.Options{
		Method:  "POST",
		URL:     srv.URL,
		JSON:    "@" + tmpFile.Name(),
		HTTP3:   true,
		Headers: []string{},
	}

	// Execute request (will fallback from HTTP/3 to HTTP/1.1)
	var out, errOut bytes.Buffer
	err = client.DoWithWriters(opts, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response contains our data
	if !strings.Contains(out.String(), "source") {
		t.Errorf("response should contain file data, got: %s", out.String())
	}
}

// TestHTTP3FallbackWithDirectString tests direct string with HTTP/3 fallback
func TestHTTP3FallbackWithDirectString(t *testing.T) {
	// Create a test server
	srv := startServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	// Create options for HTTP/3 with direct string
	opts := &cli.Options{
		Method:  "POST",
		URL:     srv.URL,
		JSON:    `{"direct":"string"}`,
		HTTP3:   true,
		Headers: []string{},
	}

	// Execute request (will fallback from HTTP/3 to HTTP/1.1)
	var out, errOut bytes.Buffer
	err := client.DoWithWriters(opts, &out, &errOut)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify response contains our data
	if !strings.Contains(out.String(), "direct") {
		t.Errorf("response should contain string data, got: %s", out.String())
	}
}

// TestFileSeekingAcrossRetries tests file seeking for multiple retries
func TestFileSeekingAcrossRetries(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "httpcat-seek-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data: "ABCDEFGHIJ"
	testData := "ABCDEFGHIJ"
	if _, err := tmpFile.WriteString(testData); err != nil {
		t.Fatalf("failed to write test data: %v", err)
	}
	tmpFile.Close()

	// Create bodySource for the file
	spec := "@" + tmpFile.Name()
	bs, err := client.NewBodySource(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer bs.Close()

	// Read multiple times and verify seeking works
	for i := 0; i < 3; i++ {
		reader, err := bs.GetReader()
		if err != nil {
			t.Fatalf("attempt %d: unexpected error: %v", i, err)
		}
		data, _ := io.ReadAll(reader)
		reader.Close()

		if string(data) != testData {
			t.Errorf("attempt %d: got %q, want %q", i, string(data), testData)
		}
	}
}

// TestBufferStdinIdempotent tests that buffering stdin multiple times is safe
func TestBufferStdinIdempotent(t *testing.T) {
	// Create a pipe with test data
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	testData := "idempotent test"
	go func() {
		w.WriteString(testData)
		w.Close()
	}()

	// Replace stdin
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	// Create bodySource for stdin
	bs, err := client.NewBodySource("@-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer bs.Close()

	// Buffer stdin multiple times (should only buffer once)
	if err := bs.BufferStdinIfNeeded(); err != nil {
		t.Fatalf("first buffer: %v", err)
	}
	if err := bs.BufferStdinIfNeeded(); err != nil {
		t.Fatalf("second buffer: %v", err)
	}

	// Verify both reads work
	reader1, _ := bs.GetReader()
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	reader2, _ := bs.GetReader()
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != testData || string(data2) != testData {
		t.Errorf("data mismatch after idempotent buffering")
	}
}

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

func TestBodySourceDirectString(t *testing.T) {
	bs, err := client.NewBodySource("hello world")
	if err != nil || bs == nil {
		t.Fatalf("error: %v", err)
	}

	reader, _ := bs.GetReader()
	data, _ := io.ReadAll(reader)
	reader.Close()

	if string(data) != "hello world" {
		t.Errorf("got %q, want %q", string(data), "hello world")
	}
	bs.Close()
}

func TestBodySourceDirectStringRetry(t *testing.T) {
	bs, _ := client.NewBodySource("test content")

	reader1, _ := bs.GetReader()
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	reader2, _ := bs.GetReader()
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != string(data2) {
		t.Errorf("reads differ: %q vs %q", string(data1), string(data2))
	}
	bs.Close()
}

func TestBodySourceEmptyString(t *testing.T) {
	bs, err := client.NewBodySource("")
	if err != nil || bs != nil {
		t.Fatal("expected nil for empty spec")
	}
}

func TestBodySourceFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "test-*.txt")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString("file content")
	tmpFile.Close()

	bs, _ := client.NewBodySource("@" + tmpFile.Name())
	defer bs.Close()

	reader1, _ := bs.GetReader()
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	reader2, _ := bs.GetReader()
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != "file content" || string(data2) != "file content" {
		t.Errorf("file content mismatch")
	}
}

func TestBodySourceFileNotFound(t *testing.T) {
	bs, err := client.NewBodySource("@/nonexistent/file")
	if err == nil || bs != nil {
		t.Fatal("expected error for missing file")
	}
}

func TestBodySourceLargeFile(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "large-*.bin")
	defer os.Remove(tmpFile.Name())

	chunk := make([]byte, 1024*1024) // 1MB
	for i := 0; i < 5; i++ {         // 5MB total
		tmpFile.Write(chunk)
	}
	tmpFile.Close()

	bs, _ := client.NewBodySource("@" + tmpFile.Name())
	defer bs.Close()

	reader, _ := bs.GetReader()
	defer reader.Close()

	data := make([]byte, 1024)
	n, _ := reader.Read(data)
	if n == 0 {
		t.Fatal("failed to read large file")
	}
}

func TestBufferStdinIfNeeded(t *testing.T) {
	r, w, _ := os.Pipe()
	defer r.Close()

	go func() {
		w.WriteString("stdin test data")
		w.Close()
	}()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	bs, _ := client.NewBodySource("@-")
	defer bs.Close()

	bs.BufferStdinIfNeeded()

	reader1, _ := bs.GetReader()
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	reader2, _ := bs.GetReader()
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != "stdin test data" || string(data2) != "stdin test data" {
		t.Errorf("stdin data mismatch")
	}
}

func TestStdinRetryWithoutBuffer(t *testing.T) {
	r, w, _ := os.Pipe()
	go func() {
		w.WriteString("data")
		w.Close()
	}()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	bs, _ := client.NewBodySource("@-")
	defer bs.Close()

	reader1, _ := bs.GetReader()
	io.ReadAll(reader1)
	reader1.Close()

	_, err := bs.GetReader()
	if err == nil || !strings.Contains(err.Error(), "cannot retry") {
		t.Errorf("expected retry error, got %v", err)
	}
}

func TestHTTP3FallbackWithStdin(t *testing.T) {
	r, w, _ := os.Pipe()
	go func() {
		w.WriteString(`{"test":"data"}`)
		w.Close()
	}()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	srv := startServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	opts := &cli.Options{
		Method:  "POST",
		URL:     srv.URL,
		JSON:    "@-",
		HTTP3:   true,
		Headers: []string{},
	}

	var out, errOut bytes.Buffer
	err := client.DoWithWriters(opts, &out, &errOut)
	if err != nil {
		t.Fatalf("HTTP/3 fallback: %v", err)
	}

	if !strings.Contains(out.String(), "test") {
		t.Errorf("response missing data")
	}
}

func TestHTTP3FallbackWithFile(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "test-*.json")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(`{"source":"file"}`)
	tmpFile.Close()

	srv := startServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	opts := &cli.Options{
		Method:  "POST",
		URL:     srv.URL,
		JSON:    "@" + tmpFile.Name(),
		HTTP3:   true,
		Headers: []string{},
	}

	var out, errOut bytes.Buffer
	if err := client.DoWithWriters(opts, &out, &errOut); err != nil {
		t.Fatalf("HTTP/3 fallback: %v", err)
	}

	if !strings.Contains(out.String(), "source") {
		t.Errorf("response missing data")
	}
}

func TestHTTP3FallbackWithDirectString(t *testing.T) {
	srv := startServer(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	opts := &cli.Options{
		Method:  "POST",
		URL:     srv.URL,
		JSON:    `{"direct":"string"}`,
		HTTP3:   true,
		Headers: []string{},
	}

	var out, errOut bytes.Buffer
	if err := client.DoWithWriters(opts, &out, &errOut); err != nil {
		t.Fatalf("HTTP/3 fallback: %v", err)
	}

	if !strings.Contains(out.String(), "direct") {
		t.Errorf("response missing data")
	}
}

func TestFileSeekingMultipleReads(t *testing.T) {
	tmpFile, _ := os.CreateTemp("", "seek-*.bin")
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("ABCDEFGHIJ")
	tmpFile.Close()

	bs, _ := client.NewBodySource("@" + tmpFile.Name())
	defer bs.Close()

	for i := 0; i < 3; i++ {
		reader, err := bs.GetReader()
		if err != nil {
			t.Fatalf("attempt %d: %v", i, err)
		}
		data, _ := io.ReadAll(reader)
		reader.Close()

		if string(data) != "ABCDEFGHIJ" {
			t.Errorf("attempt %d: got %q", i, string(data))
		}
	}
}

func TestBufferStdinIdempotent(t *testing.T) {
	r, w, _ := os.Pipe()
	go func() {
		w.WriteString("idempotent")
		w.Close()
	}()

	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	bs, _ := client.NewBodySource("@-")
	defer bs.Close()

	bs.BufferStdinIfNeeded()
	bs.BufferStdinIfNeeded()

	reader1, _ := bs.GetReader()
	data1, _ := io.ReadAll(reader1)
	reader1.Close()

	reader2, _ := bs.GetReader()
	data2, _ := io.ReadAll(reader2)
	reader2.Close()

	if string(data1) != "idempotent" || string(data2) != "idempotent" {
		t.Errorf("data mismatch after idempotent buffering")
	}
}

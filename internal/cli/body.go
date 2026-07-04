package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// ReadBody reads the request body from options.
// Supports:
//   - Direct string: "hello world"
//   - stdin: "@-"
//   - file: "@/path/to/file"
//
// Returns io.ReadCloser or error if reading fails.
func ReadBody(opts *Options) (io.ReadCloser, error) {
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
		return io.NopCloser(os.Stdin), nil
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

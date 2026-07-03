package version_test

import (
	"strings"
	"testing"

	"github.com/aaronriekenberg/httpcat/internal/version"
)

func TestStringContainsHttpcat(t *testing.T) {
	s := version.String()
	if !strings.HasPrefix(s, "httpcat ") {
		t.Errorf("version string should start with 'httpcat ', got %q", s)
	}
}

func TestStringContainsVersion(t *testing.T) {
	s := version.String()
	if !strings.Contains(s, version.Version) {
		t.Errorf("version string %q should contain Version %q", s, version.Version)
	}
}

func TestStringContainsCommit(t *testing.T) {
	s := version.String()
	if !strings.Contains(s, version.Commit) {
		t.Errorf("version string %q should contain Commit %q", s, version.Commit)
	}
}

func TestStringContainsDate(t *testing.T) {
	s := version.String()
	if !strings.Contains(s, version.Date) {
		t.Errorf("version string %q should contain Date %q", s, version.Date)
	}
}

func TestDefaultVersion(t *testing.T) {
	// When not injected via ldflags, defaults should be non-empty sentinels.
	if version.Version == "" {
		t.Error("Version should not be empty")
	}
	if version.Commit == "" {
		t.Error("Commit should not be empty")
	}
	if version.Date == "" {
		t.Error("Date should not be empty")
	}
}

// Package version holds build-time version information injected via ldflags.
package version

import "fmt"

// These variables are set at build time with:
//
//	-ldflags "-X github.com/aaronriekenberg/httpcat/internal/version.Version=v1.2.3
//	          -X github.com/aaronriekenberg/httpcat/internal/version.Commit=abc1234
//	          -X github.com/aaronriekenberg/httpcat/internal/version.Date=2026-07-03"
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return fmt.Sprintf("httpcat %s (commit %s, built %s)", Version, Commit, Date)
}

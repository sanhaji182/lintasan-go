// Package version is the single source of truth for the Lintasan build version.
//
// The default value tracks the latest release tag. It can be overridden at
// build time via ldflags so binaries report the exact tag/commit they were
// cut from, e.g.:
//
//	go build -ldflags="-X github.com/sanhaji182/lintasan-go/internal/version.Version=v2.4.0"
//
// The Makefile wires this automatically from `git describe --tags`.
package version

// Version is the server version reported by /health, /metrics build_info, and
// the MCP server_info tool. Overridable at link time (see package doc).
var Version = "2.4.0"

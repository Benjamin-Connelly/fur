// Package remote provides SSH/SFTP remote file access for fur.
// It implements a sync/cache model: files are downloaded from the remote
// host to a local cache directory, and a background goroutine polls for
// changes periodically.
package remote

import (
	"fmt"
	"regexp"
	"strings"
)

// Target represents a parsed remote path specification.
type Target struct {
	User string // SSH username (empty = use ~/.ssh/config or current user)
	Host string // hostname or SSH config alias
	Port int    // SSH port (0 = use default 22 or ~/.ssh/config)
	Path string // remote directory path
}

// String returns the SCP-style representation.
func (t Target) String() string {
	var b strings.Builder
	if t.User != "" {
		b.WriteString(t.User)
		b.WriteString("@")
	}
	b.WriteString(t.Host)
	if t.Port != 0 && t.Port != 22 {
		fmt.Fprintf(&b, ":%d", t.Port)
	}
	b.WriteString(":")
	b.WriteString(t.Path)
	return b.String()
}

// Display returns a short display string for the status bar.
func (t Target) Display() string {
	var b strings.Builder
	if t.User != "" {
		b.WriteString(t.User)
		b.WriteString("@")
	}
	b.WriteString(t.Host)
	b.WriteString(":")
	b.WriteString(t.Path)
	return b.String()
}

// scpPattern matches SCP-style remote paths:
//
//	user@host:/path
//	host:/path
//	user@host:port:/path
var scpPattern = regexp.MustCompile(`^(?:([^@:]+)@)?([^:]+):(\d+:)?(.+)$`)

// ParseTarget parses an SCP-style remote path specification.
// Supported formats:
//   - host:/path
//   - user@host:/path
//   - user@host:port:/path
//
// Returns nil if the input is not a remote path (no ':' separator
// or looks like a Windows drive letter).
func ParseTarget(s string) *Target {
	// Not a remote path if no colon or looks like a Windows drive (C:\)
	if !strings.Contains(s, ":") {
		return nil
	}

	// Windows drive letter check: single letter followed by :\ or :/
	if len(s) >= 2 && s[1] == ':' && (len(s) == 2 || s[2] == '/' || s[2] == '\\') {
		return nil
	}

	m := scpPattern.FindStringSubmatch(s)
	if m == nil {
		return nil
	}

	path := m[4]
	// Reject paths that are just punctuation artifacts (e.g. "host::" → path=":")
	if path == ":" || path == "" {
		return nil
	}

	t := &Target{
		User: m[1],
		Host: m[2],
		Path: path,
	}

	// Parse optional port
	if m[3] != "" {
		portStr := strings.TrimSuffix(m[3], ":")
		fmt.Sscanf(portStr, "%d", &t.Port)
	}

	// Default to home directory if path is empty
	if t.Path == "" {
		t.Path = "."
	}

	return t
}

// IsRemotePath returns true if the string looks like a remote path spec.
func IsRemotePath(s string) bool {
	return ParseTarget(s) != nil
}

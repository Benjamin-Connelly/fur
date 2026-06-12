package remote

import (
	"strings"
	"testing"
)

// FuzzParseTarget fuzzes the SCP-style "host:path" / "user@host:path" parser.
// Invariant: never panics on arbitrary input, and when it returns a target the
// host carries no whitespace/control bytes that would later be handed to a
// dialer or SSH config lookup.
func FuzzParseTarget(f *testing.F) {
	seeds := []string{
		"host:/path",
		"user@host:/path",
		"host:22:/path",
		"@named",
		"-oProxyCommand=evil@host:/x",
		"host:/path with spaces",
		"[::1]:/path",
		"a@b@c:/d",
		strings.Repeat("a", 70000) + ":/x",
		"",
		":",
		"::::",
		"\x00host:/p",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		tgt := ParseTarget(s)
		if tgt == nil {
			return
		}
		if strings.ContainsAny(tgt.Host, " \t\n\r\x00") {
			t.Errorf("ParseTarget(%q) yielded host with whitespace/control bytes: %q", s, tgt.Host)
		}
		if tgt.Port < 0 {
			t.Errorf("ParseTarget(%q) yielded negative port %d", s, tgt.Port)
		}
		// IsRemotePath must not panic either.
		_ = IsRemotePath(s)
	})
}

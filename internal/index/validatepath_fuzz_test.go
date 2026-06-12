package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzValidatePath is the chokepoint fuzz for Index.ValidatePath
// (lookit-9py.3.3). ValidatePath is the shared path-security gate the web
// handlers delegate to; if it ever returns a path outside the index root, the
// whole document/code/raw surface becomes a traversal primitive.
//
// The invariant: for ANY input, ValidatePath either errors, or returns a
// cleaned absolute path that — after symlink resolution — is the root itself
// or lies beneath it. The fuzz corpus seeds traversal, encoding, absolute,
// and control-character payloads; the body asserts containment under a
// synthetic root populated with a real in-root file and a symlink escaping it.
func FuzzValidatePath(f *testing.F) {
	seeds := []string{
		"README.md",
		"../etc/passwd",
		"../../../../etc/passwd",
		"..",
		"./../x",
		"foo/../../bar",
		"/etc/passwd",
		"/abs/path",
		"sub/file.md",
		"%2e%2e/%2e%2e/etc/passwd",
		"..\\..\\windows",
		"a/./b/../c",
		"\x00/etc/passwd",
		"esc", // the escaping symlink created below
		"in.md",
		strings.Repeat("../", 64) + "etc/passwd",
		"",
		".",
		"./",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	root := f.TempDir()
	outside := f.TempDir()
	// in-root file
	if err := os.WriteFile(filepath.Join(root, "in.md"), []byte("ok"), 0o600); err != nil {
		f.Fatalf("write in-root: %v", err)
	}
	// secret outside root + a symlink inside root pointing at it
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("SECRET"), 0o600); err != nil {
		f.Fatalf("write secret: %v", err)
	}
	_ = os.Symlink(secret, filepath.Join(root, "esc")) // best-effort; skipped on platforms without symlinks

	// Resolve the root the way ValidatePath does, so the containment check uses
	// the same canonical form (e.g. /tmp may be a symlink to /private/tmp).
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		f.Fatalf("eval root: %v", err)
	}

	idx := New(root)

	f.Fuzz(func(t *testing.T, rel string) {
		got, err := idx.ValidatePath(rel)
		if err != nil {
			return // rejection is always acceptable
		}
		// On success the returned path must be contained in the root.
		if got != resolvedRoot && !strings.HasPrefix(got, resolvedRoot+string(filepath.Separator)) {
			t.Errorf("ValidatePath(%q) returned %q which escapes root %q", rel, got, resolvedRoot)
		}
		// And the returned path must actually exist and resolve within root
		// (no TOCTOU gap: ValidatePath returns the resolved path).
		final, evErr := filepath.EvalSymlinks(got)
		if evErr != nil {
			return
		}
		if final != resolvedRoot && !strings.HasPrefix(final, resolvedRoot+string(filepath.Separator)) {
			t.Errorf("ValidatePath(%q) returned %q whose resolved target %q escapes root %q", rel, got, final, resolvedRoot)
		}
	})
}

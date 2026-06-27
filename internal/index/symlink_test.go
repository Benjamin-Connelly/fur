package index

import (
	"os"
	"path/filepath"
	"testing"
)

// TestBuild_SymlinkEscapeNotIndexed is the Chain B regression guard.
//
// Under the audit threat model a directory adversary controls the tree a
// victim browses with fur. The adversary plants a symlink inside the root
// whose target is a sensitive file outside the root (e.g.
// notes.md -> ~/.ssh/id_rsa). Build uses Lstat, so the symlink is indexed as
// a regular entry; the TUI preview and `fur serve` then open entry.Path,
// follow the link, and disclose the out-of-root target.
//
// The fix contains symlinks: an entry whose resolved target escapes the root
// is dropped from the index unless Options.FollowSymlinks is set. Red against
// master (which indexed the escaping symlink). References lookit-9py.3.6 /
// .4.7; SECURITY-INVENTORY symlink surface.
func TestBuild_SymlinkEscapeNotIndexed(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()

	secret := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secret, []byte("PRIVATE KEY MATERIAL"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	// In-root legitimate file, to prove containment is selective.
	if err := os.WriteFile(filepath.Join(root, "real.md"), []byte("# real"), 0o644); err != nil {
		t.Fatalf("write real.md: %v", err)
	}
	// Hostile symlink escaping the root.
	if err := os.Symlink(secret, filepath.Join(root, "notes.md")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	idx := New(root)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}

	for _, e := range idx.Entries() {
		if e.RelPath == "notes.md" {
			t.Errorf("escaping symlink notes.md -> %s was indexed (Chain B); "+
				"path=%s — escaping symlinks must be contained", secret, e.Path)
		}
	}
	if idx.Lookup("real.md") == nil {
		t.Error("in-root real.md was dropped; containment must be selective")
	}
}

// TestBuild_SymlinkEscapeOptIn confirms FollowSymlinks restores the old
// behavior for operators who explicitly opt in.
func TestBuild_SymlinkEscapeOptIn(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(secret, []byte("x"), 0o600); err != nil {
		t.Fatalf("write secret: %v", err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "notes.md")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	idx := NewWithOptions(root, Options{FollowSymlinks: true})
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if idx.Lookup("notes.md") == nil {
		t.Error("with FollowSymlinks=true the escaping symlink should be indexed")
	}
}

// TestBuild_SymlinkWithinRootIndexed confirms an in-root symlink (target
// stays under root) is still surfaced — containment only blocks escapes.
func TestBuild_SymlinkWithinRootIndexed(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.md")
	if err := os.WriteFile(target, []byte("# t"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(root, "alias.md")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	idx := New(root)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	if idx.Lookup("alias.md") == nil {
		t.Error("in-root symlink alias.md should be indexed")
	}
}

// TestValidatePath_ReturnsResolvedPath confirms ValidatePath returns the
// symlink-resolved path (not the unresolved join), so callers open the bytes
// that were validated and the second symlink follow at open time cannot
// diverge from the check.
func TestValidatePath_ReturnsResolvedPath(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real.md")
	if err := os.WriteFile(target, []byte("# r"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	if err := os.Symlink(target, filepath.Join(root, "alias.md")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	idx := New(root)
	got, err := idx.ValidatePath("alias.md")
	if err != nil {
		t.Fatalf("ValidatePath: %v", err)
	}
	want, _ := filepath.EvalSymlinks(target)
	if got != want {
		t.Errorf("ValidatePath returned %q, want resolved %q", got, want)
	}
}

// TestValidatePath_RejectsTraversal and escape are the core chokepoint
// guarantees relied on by every web route.
func TestValidatePath_RejectsTraversal(t *testing.T) {
	idx := New(t.TempDir())
	if _, err := idx.ValidatePath("../../etc/passwd"); err == nil {
		t.Error("ValidatePath accepted ../ traversal")
	}
}

func TestValidatePath_RejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("x"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.Symlink(secret, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	idx := New(root)
	if _, err := idx.ValidatePath("escape"); err == nil {
		t.Error("ValidatePath accepted a symlink escaping the root")
	}
}

// TestValidatePath_SymlinkedRoot guards the case where the served root is
// itself reached through a symlink (e.g. ~/.claude/skills -> ~/dotfiles/...).
// EvalSymlinks resolves a requested file to a path under the symlink TARGET, so
// comparing it to the raw root used to look like an escape and 404 every file
// even though the directory listing showed it.
func TestValidatePath_SymlinkedRoot(t *testing.T) {
	// Mirror the real case: a symlinked PARENT (~/.claude/skills -> ~/dotfiles)
	// with the served root a real directory reached through it.
	real := t.TempDir()
	sub := filepath.Join(real, "security-scan")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "skill.md"), []byte("# skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "skills")
	if err := os.Symlink(real, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	idx := New(filepath.Join(link, "security-scan")) // root reached via the symlink
	idx.Build()

	if idx.Lookup("skill.md") == nil {
		t.Fatal("skill.md should be indexed under a symlinked root")
	}
	if _, err := idx.ValidatePath("skill.md"); err != nil {
		t.Errorf("ValidatePath should pass for a file under a symlinked root, got %v", err)
	}
}

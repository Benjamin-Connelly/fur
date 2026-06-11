package index

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestFulltextCacheDirPerms is the Chain F / Chain H regression guard.
//
// The Bleve fulltext index mirrors the content of every markdown file the
// user has browsed. It lives at cacheDir/index.bleve. Before the fix the
// cache directory was created 0o755 (and Bleve's store inherited the process
// umask), so on a shared/multi-user box another tenant could traverse into
// the directory and read the indexed content of the victim's private files
// — cross-session, cross-user disclosure.
//
// The fix clamps the cache directory and the index tree to owner-only
// (0700 dirs, 0600 files). References lookit-9py.3.10 / .3.12 / .4.6.
func TestFulltextCacheDirPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission model; audit targets linux/darwin")
	}
	cacheDir := filepath.Join(t.TempDir(), "fur-cache")

	ft, err := NewFulltextIndex(cacheDir)
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	info, err := os.Stat(cacheDir)
	if err != nil {
		t.Fatalf("stat cacheDir: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("cache dir mode = %o, want owner-only (no group/other bits); "+
			"group/other can read the browsed-content index (Chain F/H)", perm)
	}

	// The Bleve index tree must also be owner-only.
	indexPath := filepath.Join(cacheDir, "index.bleve")
	err = filepath.WalkDir(indexPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return nil
		}
		if fi.Mode().Perm()&0o077 != 0 {
			t.Errorf("index path %s mode = %o, want owner-only", path, fi.Mode().Perm())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk index: %v", err)
	}
}

// TestFulltextCacheReopenTightensPerms confirms that reopening an index left
// world-readable by an older fur re-clamps the perms.
func TestFulltextCacheReopenTightensPerms(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permission model")
	}
	cacheDir := filepath.Join(t.TempDir(), "fur-cache")

	ft, err := NewFulltextIndex(cacheDir)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	ft.Close()

	// Simulate a loose-perm cache from an older fur.
	if err := os.Chmod(cacheDir, 0o755); err != nil {
		t.Fatalf("chmod loose: %v", err)
	}

	ft2, err := NewFulltextIndex(cacheDir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer ft2.Close()

	info, _ := os.Stat(cacheDir)
	if info.Mode().Perm()&0o077 != 0 {
		t.Errorf("reopened cache dir mode = %o, want owner-only", info.Mode().Perm())
	}
}

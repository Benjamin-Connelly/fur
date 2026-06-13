package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gogit "github.com/go-git/go-git/v5"
	gogitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// setupRepoWithRemote builds a temp repo with one committed file and an
// origin remote, then opens it. Used to exercise the permalink/remote-URL
// builders end to end (lookit-9py.2.1 coverage for internal/git).
func setupRepoWithRemote(t *testing.T, remoteURL string) (*Repo, string) {
	t.Helper()
	dir := t.TempDir()

	repo, err := gogit.PlainInit(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateRemote(&gogitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{remoteURL},
	}); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wt, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Add("README.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := wt.Commit("init", &gogit.CommitOptions{
		Author: &object.Signature{Name: "T", Email: "t@t.io", When: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	repoCacheMu.Lock()
	delete(repoCache, dir)
	repoCacheMu.Unlock()

	r, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return r, dir
}

func TestCurrentRemoteURL(t *testing.T) {
	r, _ := setupRepoWithRemote(t, "git@github.com:owner/repo.git")
	got, err := r.CurrentRemoteURL()
	if err != nil {
		t.Fatalf("CurrentRemoteURL: %v", err)
	}
	if got != "git@github.com:owner/repo.git" {
		t.Errorf("CurrentRemoteURL = %q", got)
	}
}

func TestPermalinkBuilders(t *testing.T) {
	cases := []struct {
		name     string
		remote   string
		wantHost string
		wantBlob string // path segment between repo and ref
	}{
		{"github ssh", "git@github.com:owner/repo.git", "github.com/owner/repo", "/blob/"},
		{"github https", "https://github.com/owner/repo.git", "github.com/owner/repo", "/blob/"},
		{"gitlab", "git@gitlab.com:owner/repo.git", "gitlab.com/owner/repo", "/-/blob/"},
		{"codeberg", "https://codeberg.org/owner/repo", "codeberg.org/owner/repo", "/src/"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, _ := setupRepoWithRemote(t, tc.remote)

			link, err := r.Permalink("README.md", 1)
			if err != nil {
				t.Fatalf("Permalink: %v", err)
			}
			if !strings.Contains(link, tc.wantHost) {
				t.Errorf("Permalink %q missing host %q", link, tc.wantHost)
			}
			if !strings.Contains(link, tc.wantBlob) {
				t.Errorf("Permalink %q missing blob segment %q", link, tc.wantBlob)
			}
			if !strings.Contains(link, "README.md") {
				t.Errorf("Permalink %q missing file path", link)
			}

			branchLink, err := r.PermalinkForBranch("README.md", "main", 5)
			if err != nil {
				t.Fatalf("PermalinkForBranch: %v", err)
			}
			if !strings.Contains(branchLink, "main") {
				t.Errorf("PermalinkForBranch %q missing branch ref", branchLink)
			}

			rangeLink, err := r.PermalinkForRange("README.md", 3, 8)
			if err != nil {
				t.Fatalf("PermalinkForRange: %v", err)
			}
			if !strings.Contains(rangeLink, "README.md") {
				t.Errorf("PermalinkForRange %q missing file", rangeLink)
			}

			fileURL, err := r.FileURL("README.md")
			if err != nil {
				t.Fatalf("FileURL: %v", err)
			}
			if !strings.Contains(fileURL, tc.wantHost) {
				t.Errorf("FileURL %q missing host", fileURL)
			}

			// CopyPermalink returns the link even when no clipboard is
			// available (headless CI), so it must not error.
			cp, err := r.CopyPermalink("README.md", 2)
			if err != nil {
				t.Fatalf("CopyPermalink: %v", err)
			}
			if cp == "" {
				t.Error("CopyPermalink returned empty link")
			}
		})
	}
}

func TestPermalink_NoRemote(t *testing.T) {
	// setupTestRepo (from git_test.go) creates a repo with no remote.
	r, _ := setupTestRepo(t)
	if _, err := r.Permalink("README.md", 1); err == nil {
		t.Error("Permalink should error when there is no origin remote")
	}
}

func TestFileStatusAt(t *testing.T) {
	r, dir := setupRepoWithRemote(t, "git@github.com:owner/repo.git")

	// Clean, committed file.
	st, err := r.FileStatusAt("README.md")
	if err != nil {
		t.Fatalf("FileStatusAt(clean): %v", err)
	}
	if st.Path != "README.md" {
		t.Errorf("FileStatusAt path = %q, want README.md", st.Path)
	}

	// Modify it → status should reflect a worktree change.
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repoCacheMu.Lock()
	delete(repoCache, dir)
	repoCacheMu.Unlock()
	r2, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r2.FileStatusAt("README.md"); err != nil {
		t.Fatalf("FileStatusAt(modified): %v", err)
	}
}

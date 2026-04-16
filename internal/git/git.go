package git

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

var errStopIteration = errors.New("stop iteration")

// Repo wraps a go-git repository with convenience methods.
type Repo struct {
	repo     *gogit.Repository
	root     string
	worktree *gogit.Worktree
}

// FileStatus represents the git status of a single file.
type FileStatus struct {
	Path     string
	Staging  StatusCode
	Worktree StatusCode
}

// CommitInfo holds metadata for a single commit.
type CommitInfo struct {
	Hash    string
	Author  string
	Date    time.Time
	Message string
}

// StatusCode maps to go-git status codes.
type StatusCode byte

const (
	Unmodified StatusCode = ' '
	Modified   StatusCode = 'M'
	Added      StatusCode = 'A'
	Deleted    StatusCode = 'D'
	Renamed    StatusCode = 'R'
	Copied     StatusCode = 'C'
	Untracked  StatusCode = '?'
)

var (
	repoCache   = make(map[string]*Repo)
	repoCacheMu sync.Mutex
)

// Open opens the git repository at or above the given path.
// Repeated calls for the same absolute path return a cached Repo.
func Open(path string) (*Repo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving path %q: %w", path, err)
	}

	repoCacheMu.Lock()
	if cached, ok := repoCache[absPath]; ok {
		repoCacheMu.Unlock()
		return cached, nil
	}
	repoCacheMu.Unlock()

	repo, err := gogit.PlainOpenWithOptions(absPath, &gogit.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("opening repo at %q: %w", absPath, err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("getting worktree for %q: %w", absPath, err)
	}

	r := &Repo{
		repo:     repo,
		root:     absPath,
		worktree: wt,
	}

	repoCacheMu.Lock()
	repoCache[absPath] = r
	repoCacheMu.Unlock()

	return r, nil
}

// IsRepo returns true if path is inside a git repository.
func IsRepo(path string) bool {
	_, err := Open(path)
	return err == nil
}

// Root returns the repository root path.
func (r *Repo) Root() string {
	return r.root
}

// Branch returns the current branch name.
func (r *Repo) Branch() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("reading HEAD: %w", err)
	}
	return head.Name().Short(), nil
}

// BranchList returns all local branch names.
func (r *Repo) BranchList() ([]string, error) {
	iter, err := r.repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("listing branches: %w", err)
	}

	var names []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		names = append(names, ref.Name().Short())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating branches: %w", err)
	}
	return names, nil
}

// Status returns the working tree status for all files.
func (r *Repo) Status() ([]FileStatus, error) {
	status, err := r.worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("reading worktree status: %w", err)
	}

	var files []FileStatus
	for path, s := range status {
		files = append(files, FileStatus{
			Path:     path,
			Staging:  StatusCode(s.Staging),
			Worktree: StatusCode(s.Worktree),
		})
	}
	return files, nil
}

// FileStatusAt returns the git status for a specific file path relative to the repo root.
func (r *Repo) FileStatusAt(relPath string) (FileStatus, error) {
	status, err := r.worktree.Status()
	if err != nil {
		return FileStatus{}, fmt.Errorf("reading worktree status for %q: %w", relPath, err)
	}

	s := status.File(relPath)
	return FileStatus{
		Path:     relPath,
		Staging:  StatusCode(s.Staging),
		Worktree: StatusCode(s.Worktree),
	}, nil
}

// IsClean returns true if the worktree has no modifications.
func (r *Repo) IsClean() (bool, error) {
	status, err := r.worktree.Status()
	if err != nil {
		return false, fmt.Errorf("reading worktree status: %w", err)
	}
	return status.IsClean(), nil
}

// Log returns the last n commits from HEAD.
func (r *Repo) Log(n int) ([]CommitInfo, error) {
	iter, err := r.repo.Log(&gogit.LogOptions{})
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}

	var commits []CommitInfo
	count := 0
	err = iter.ForEach(func(c *object.Commit) error {
		if count >= n {
			return errStopIteration
		}
		commits = append(commits, CommitInfo{
			Hash:    c.Hash.String(),
			Author:  c.Author.Name,
			Date:    c.Author.When,
			Message: c.Message,
		})
		count++
		return nil
	})
	if err != nil && !errors.Is(err, errStopIteration) {
		return nil, fmt.Errorf("iterating commits: %w", err)
	}

	return commits, nil
}

// CurrentRemoteURL returns the fetch URL for the "origin" remote.
func (r *Repo) CurrentRemoteURL() (string, error) {
	remotes, err := r.Remotes()
	if err != nil {
		return "", err
	}
	url, ok := remotes["origin"]
	if !ok {
		return "", fmt.Errorf("no origin remote configured")
	}
	return url, nil
}

// Remotes returns the list of remote names and URLs.
func (r *Repo) Remotes() (map[string]string, error) {
	remotes, err := r.repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("listing remotes: %w", err)
	}

	result := make(map[string]string)
	for _, remote := range remotes {
		cfg := remote.Config()
		if len(cfg.URLs) > 0 {
			result[cfg.Name] = cfg.URLs[0]
		}
	}
	return result, nil
}

// HeadHash returns the HEAD commit hash.
func (r *Repo) HeadHash() (string, error) {
	head, err := r.repo.Head()
	if err != nil {
		return "", fmt.Errorf("reading HEAD hash: %w", err)
	}
	return head.Hash().String(), nil
}

// Tags returns all tag names.
func (r *Repo) Tags() ([]string, error) {
	tags, err := r.repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	var names []string
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		names = append(names, ref.Name().Short())
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating tags: %w", err)
	}
	return names, err
}

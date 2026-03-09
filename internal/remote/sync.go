package remote

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
)

// SyncState represents the current sync status.
type SyncState int

const (
	SyncIdle SyncState = iota
	SyncRunning
	SyncError
)

// SyncStatus reports the current state of the sync layer.
type SyncStatus struct {
	State      SyncState
	LastSync   time.Time
	FilesTotal int
	LastError  error
}

// Syncer downloads remote files to a local cache directory and
// periodically polls for changes.
type Syncer struct {
	conn     *Conn
	cacheDir string // local cache root (e.g. ~/.cache/lookit/remote/host/path)
	remotePath string

	status   SyncStatus
	mu       sync.RWMutex

	// mtimeCache tracks remote file modification times to detect changes
	mtimeCache map[string]time.Time

	pollInterval time.Duration
	done         chan struct{}
	onChange     func() // called when files change during poll
}

// NewSyncer creates a syncer that caches remote files locally.
func NewSyncer(conn *Conn, cacheDir string) *Syncer {
	return &Syncer{
		conn:         conn,
		cacheDir:     cacheDir,
		remotePath:   conn.Target().Path,
		mtimeCache:   make(map[string]time.Time),
		pollInterval: 15 * time.Second,
		done:         make(chan struct{}),
	}
}

// CacheDir returns the local cache directory path.
func (s *Syncer) CacheDir() string {
	return s.cacheDir
}

// Status returns the current sync status.
func (s *Syncer) Status() SyncStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

// SetOnChange registers a callback invoked when remote files change.
func (s *Syncer) SetOnChange(fn func()) {
	s.onChange = fn
}

// InitialSync performs the first full download of remote files.
func (s *Syncer) InitialSync() error {
	s.mu.Lock()
	s.status.State = SyncRunning
	s.mu.Unlock()

	err := s.doSync()

	s.mu.Lock()
	if err != nil {
		s.status.State = SyncError
		s.status.LastError = err
	} else {
		s.status.State = SyncIdle
		s.status.LastSync = time.Now()
		s.status.LastError = nil
	}
	s.mu.Unlock()

	return err
}

// StartPolling begins background polling for remote changes.
func (s *Syncer) StartPolling() {
	go s.pollLoop()
}

// Stop halts the background polling.
func (s *Syncer) Stop() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
}

func (s *Syncer) pollLoop() {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if s.conn.State() != ConnConnected {
				continue
			}

			changed, err := s.pollChanges()
			if err != nil {
				log.Printf("remote sync poll: %v", err)
				s.mu.Lock()
				s.status.LastError = err
				s.mu.Unlock()
				continue
			}

			if changed && s.onChange != nil {
				s.onChange()
			}

			s.mu.Lock()
			s.status.LastSync = time.Now()
			s.status.LastError = nil
			s.mu.Unlock()

		case <-s.done:
			return
		}
	}
}

// doSync walks the remote tree and downloads all files.
func (s *Syncer) doSync() error {
	client := s.conn.SFTP()
	if client == nil {
		return fmt.Errorf("not connected")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(s.cacheDir, 0o755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}

	// Walk remote tree
	fileCount := 0
	err := s.walkRemote(client, s.remotePath, func(remotePath string, info os.FileInfo) error {
		rel, err := filepath.Rel(s.remotePath, remotePath)
		if err != nil {
			return nil
		}
		if rel == "." {
			return nil
		}

		localPath := filepath.Join(s.cacheDir, rel)

		if info.IsDir() {
			return os.MkdirAll(localPath, 0o755)
		}

		// Download file
		if err := s.downloadFile(client, remotePath, localPath); err != nil {
			log.Printf("remote sync: download %s: %v", rel, err)
			return nil // continue with other files
		}

		// Cache mtime
		s.mu.Lock()
		s.mtimeCache[rel] = info.ModTime()
		s.mu.Unlock()

		fileCount++
		return nil
	})

	s.mu.Lock()
	s.status.FilesTotal = fileCount
	s.mu.Unlock()

	return err
}

// pollChanges checks for remote file modifications by comparing directory mtimes.
func (s *Syncer) pollChanges() (bool, error) {
	client := s.conn.SFTP()
	if client == nil {
		return false, fmt.Errorf("not connected")
	}

	changed := false
	err := s.walkRemote(client, s.remotePath, func(remotePath string, info os.FileInfo) error {
		rel, err := filepath.Rel(s.remotePath, remotePath)
		if err != nil || rel == "." {
			return nil
		}

		if info.IsDir() {
			localPath := filepath.Join(s.cacheDir, rel)
			os.MkdirAll(localPath, 0o755)
			return nil
		}

		s.mu.RLock()
		cached, exists := s.mtimeCache[rel]
		s.mu.RUnlock()

		if !exists || !info.ModTime().Equal(cached) {
			// File is new or modified
			localPath := filepath.Join(s.cacheDir, rel)
			if err := s.downloadFile(client, remotePath, localPath); err != nil {
				log.Printf("remote sync: update %s: %v", rel, err)
				return nil
			}
			s.mu.Lock()
			s.mtimeCache[rel] = info.ModTime()
			s.mu.Unlock()
			changed = true
		}

		return nil
	})

	// Clean up deleted files
	s.cleanDeleted(client)

	return changed, err
}

// walkRemote walks the remote directory tree, skipping hidden dirs.
func (s *Syncer) walkRemote(client *sftp.Client, root string, fn func(string, os.FileInfo) error) error {
	return s.walkDir(client, root, fn)
}

func (s *Syncer) walkDir(client *sftp.Client, dir string, fn func(string, os.FileInfo) error) error {
	entries, err := client.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading remote dir %s: %w", dir, err)
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip hidden files and directories
		if strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := dir + "/" + name

		if err := fn(fullPath, entry); err != nil {
			return err
		}

		if entry.IsDir() {
			if err := s.walkDir(client, fullPath, fn); err != nil {
				// Log but continue on directory errors
				log.Printf("remote sync: walk %s: %v", fullPath, err)
			}
		}
	}

	return nil
}

// downloadFile fetches a single file from remote to local.
func (s *Syncer) downloadFile(client *sftp.Client, remotePath, localPath string) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	remote, err := client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote: %w", err)
	}
	defer remote.Close()

	local, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local: %w", err)
	}
	defer local.Close()

	if _, err := io.Copy(local, remote); err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Preserve modification time
	info, err := client.Stat(remotePath)
	if err == nil {
		os.Chtimes(localPath, info.ModTime(), info.ModTime())
	}

	return nil
}

// cleanDeleted removes local cached files that no longer exist on the remote.
func (s *Syncer) cleanDeleted(client *sftp.Client) {
	s.mu.RLock()
	cachedFiles := make(map[string]time.Time, len(s.mtimeCache))
	for k, v := range s.mtimeCache {
		cachedFiles[k] = v
	}
	s.mu.RUnlock()

	var toDelete []string
	for rel := range cachedFiles {
		remotePath := s.remotePath + "/" + rel
		if _, err := client.Stat(remotePath); err != nil {
			// File no longer exists on remote
			toDelete = append(toDelete, rel)
			localPath := filepath.Join(s.cacheDir, rel)
			os.Remove(localPath)
		}
	}

	if len(toDelete) > 0 {
		s.mu.Lock()
		for _, rel := range toDelete {
			delete(s.mtimeCache, rel)
		}
		s.mu.Unlock()
	}
}

// CachePath returns the local cache directory for a given remote target.
// Format: ~/.cache/lookit/remote/<hash>/
// Uses a hash to avoid path length issues and special characters.
func CachePath(target Target) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("determining cache dir: %w", err)
	}

	// Hash the target to get a stable, short directory name
	h := sha256.New()
	fmt.Fprintf(h, "%s@%s:%d:%s", target.User, target.Host, target.Port, target.Path)
	hash := fmt.Sprintf("%x", h.Sum(nil))[:12]

	return filepath.Join(cacheDir, "lookit", "remote", hash), nil
}

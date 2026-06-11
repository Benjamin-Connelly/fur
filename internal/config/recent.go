package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxRecentFiles = 50

// RecentFiles manages a persistent list of recently opened files.
type RecentFiles struct {
	Files []string `json:"files"`
	path  string
}

// LoadRecentFiles reads the recent files list from disk.
func LoadRecentFiles() *RecentFiles {
	r := &RecentFiles{}
	configDir, err := ConfigDir()
	if err != nil {
		return r
	}
	r.path = filepath.Join(configDir, "recent.json")

	data, err := os.ReadFile(r.path)
	if err != nil {
		return r
	}
	_ = json.Unmarshal(data, r)
	return r
}

// Add puts a file at the front of the recent list, removing duplicates.
func (r *RecentFiles) Add(path string) {
	// Remove existing occurrence
	filtered := make([]string, 0, len(r.Files))
	for _, f := range r.Files {
		if f != path {
			filtered = append(filtered, f)
		}
	}
	// Prepend
	r.Files = append([]string{path}, filtered...)
	if len(r.Files) > maxRecentFiles {
		r.Files = r.Files[:maxRecentFiles]
	}
}

// Save writes the recent files list to disk.
func (r *RecentFiles) Save() error {
	if r.path == "" {
		configDir, err := ConfigDir()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(configDir, 0o700); err != nil {
			return err
		}
		r.path = filepath.Join(configDir, "recent.json")
	}
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}
	// 0600: the recent-files list is a record of what the user has browsed —
	// not world-readable on a shared box (audit Chain F / hardening 4.6).
	// Atomic write via a temp file + rename in the same dir so a concurrent
	// reader never sees a half-written file and a symlink swap can't redirect
	// the final write.
	dir := filepath.Dir(r.path)
	tmp, err := os.CreateTemp(dir, ".recent-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op if the rename below succeeds
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, r.path)
}

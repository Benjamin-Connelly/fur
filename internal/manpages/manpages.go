// Package manpages embeds and auto-installs man pages.
package manpages

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed pages/*.1
var pages embed.FS

// Install copies embedded man pages to ~/.local/share/man/man1/
// if they are missing or from an older version. Returns the number
// of pages installed and any error.
func Install(currentVersion string) (int, error) {
	destDir, err := destDir()
	if err != nil {
		return 0, err
	}

	// Check version stamp to avoid redundant writes
	stampFile := filepath.Join(destDir, ".fur-version")
	if data, err := os.ReadFile(stampFile); err == nil && string(data) == currentVersion {
		return 0, nil // already installed for this version
	}

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return 0, fmt.Errorf("creating man dir: %w", err)
	}

	entries, err := pages.ReadDir("pages")
	if err != nil {
		return 0, fmt.Errorf("reading embedded pages: %w", err)
	}

	installed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := pages.ReadFile("pages/" + entry.Name())
		if err != nil {
			continue
		}
		dest := filepath.Join(destDir, entry.Name())
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return installed, fmt.Errorf("writing %s: %w", dest, err)
		}
		installed++
	}

	// Write version stamp
	_ = os.WriteFile(stampFile, []byte(currentVersion), 0o644)

	return installed, nil
}

func destDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "man", "man1"), nil
}

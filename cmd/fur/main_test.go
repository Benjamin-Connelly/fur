package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/spf13/cobra/doc"
)

func init() {
	cfg = config.DefaultConfig()
}

func TestResolveRoot_Directory(t *testing.T) {
	dir := t.TempDir()

	root, initialFile, err := resolveRoot([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q", root, dir)
	}
	if initialFile != "" {
		t.Errorf("initialFile = %q, want empty", initialFile)
	}
}

func TestResolveRoot_SingleFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "README.md")
	os.WriteFile(file, []byte("# Hello\n"), 0o644)

	root, initialFile, err := resolveRoot([]string{file})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != dir {
		t.Errorf("root = %q, want %q", root, dir)
	}
	if initialFile != "README.md" {
		t.Errorf("initialFile = %q, want %q", initialFile, "README.md")
	}
}

func TestResolveRoot_NonExistent(t *testing.T) {
	_, _, err := resolveRoot([]string{"/nonexistent/path"})
	if err == nil {
		t.Error("expected error for non-existent path")
	}
}

func TestResolveRoot_DefaultConfig(t *testing.T) {
	oldRoot := cfg.Root
	cfg.Root = t.TempDir()
	defer func() { cfg.Root = oldRoot }()

	root, initialFile, err := resolveRoot(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root != cfg.Root {
		t.Errorf("root = %q, want %q", root, cfg.Root)
	}
	if initialFile != "" {
		t.Errorf("initialFile = %q, want empty", initialFile)
	}
}

// stripManHeader removes the .TH line (contains version/date that change per
// build) so we can compare content only.
func stripManHeader(content string) string {
	lines := strings.Split(content, "\n")
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, ".TH ") {
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func TestManPagesUpToDate(t *testing.T) {
	embedDir := filepath.Join("..", "..", "internal", "manpages", "pages")
	if _, err := os.Stat(embedDir); err != nil {
		t.Skip("embed directory not found (running outside repo root)")
	}

	// Generate fresh man pages to a temp dir
	tmpDir := t.TempDir()
	header := &doc.GenManHeader{
		Title:   "FUR",
		Section: "1",
	}
	if err := doc.GenManTree(rootCmd, header, tmpDir); err != nil {
		t.Fatalf("gen-man failed: %v", err)
	}

	// Compare each generated page against the committed embedded page
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("reading temp dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".1") {
			continue
		}

		generated, err := os.ReadFile(filepath.Join(tmpDir, entry.Name()))
		if err != nil {
			t.Errorf("reading generated %s: %v", entry.Name(), err)
			continue
		}

		committed, err := os.ReadFile(filepath.Join(embedDir, entry.Name()))
		if err != nil {
			t.Errorf("man page %s not found in embed dir — run: go run ./cmd/fur gen-man", entry.Name())
			continue
		}

		if stripManHeader(string(generated)) != stripManHeader(string(committed)) {
			t.Errorf("man page %s is stale — run: go run ./cmd/fur gen-man", entry.Name())
		}
	}
}

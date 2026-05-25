package index

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create directory structure
	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755)

	// Create files
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("# Guide"), 0o644)
	os.WriteFile(filepath.Join(dir, "docs", "notes.markdown"), []byte("# Notes"), 0o644)
	os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "src", "util.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte("[core]"), 0o644)
	os.WriteFile(filepath.Join(dir, "node_modules", "pkg.js"), []byte("//"), 0o644)

	return dir
}

func TestBuild(t *testing.T) {
	dir := setupTestDir(t)
	idx := New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}

	entries := idx.Entries()
	if len(entries) == 0 {
		t.Fatal("expected entries, got none")
	}

	// .git should be skipped
	for _, e := range entries {
		if filepath.Base(e.Path) == "config" && filepath.Base(filepath.Dir(e.Path)) == ".git" {
			t.Error(".git directory should be skipped")
		}
	}
}

func TestLookup(t *testing.T) {
	dir := setupTestDir(t)
	idx := New(dir)
	idx.Build()

	if e := idx.Lookup("README.md"); e == nil {
		t.Error("expected to find README.md")
	}
	if e := idx.Lookup("docs/guide.md"); e == nil {
		t.Error("expected to find docs/guide.md")
	}
	if e := idx.Lookup("nonexistent.md"); e != nil {
		t.Error("expected nil for nonexistent file")
	}
}

func TestMarkdownFiles(t *testing.T) {
	dir := setupTestDir(t)
	idx := New(dir)
	idx.Build()

	md := idx.MarkdownFiles()
	if len(md) != 3 {
		t.Errorf("expected 3 markdown files, got %d", len(md))
	}
	for _, e := range md {
		if !e.IsMarkdown {
			t.Errorf("entry %s should be markdown", e.RelPath)
		}
	}
}

func TestStats(t *testing.T) {
	dir := setupTestDir(t)
	idx := New(dir)
	idx.Build()

	stats := idx.Stats()
	if stats.FileCount == 0 {
		t.Error("expected non-zero file count")
	}
	if stats.DirCount == 0 {
		t.Error("expected non-zero dir count")
	}
}

func TestRebuild(t *testing.T) {
	dir := setupTestDir(t)
	idx := New(dir)
	idx.Build()

	before := len(idx.Entries())

	// Add a file
	os.WriteFile(filepath.Join(dir, "new.md"), []byte("# New"), 0o644)
	idx.Rebuild()

	after := len(idx.Entries())
	if after != before+1 {
		t.Errorf("expected %d entries after rebuild, got %d", before+1, after)
	}
}

func TestShowHidden(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "visible"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".hidden_dir"), 0o755)
	os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# r"), 0o644)
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("foo"), 0o644)
	os.WriteFile(filepath.Join(dir, ".hidden_dir", "note.md"), []byte("# n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".git", "config"), []byte(""), 0o644)

	t.Run("default hides dotfiles and dotdirs", func(t *testing.T) {
		idx := New(dir)
		if err := idx.Build(); err != nil {
			t.Fatalf("Build: %v", err)
		}
		if idx.Lookup(".gitignore") != nil {
			t.Error(".gitignore should be hidden by default")
		}
		if idx.Lookup(".hidden_dir") != nil {
			t.Error(".hidden_dir should be hidden by default")
		}
		if idx.Lookup(filepath.Join(".hidden_dir", "note.md")) != nil {
			t.Error("files inside .hidden_dir should not be indexed by default")
		}
		if idx.Lookup("README.md") == nil {
			t.Error("README.md should always be indexed")
		}
	})

	t.Run("ShowHidden surfaces dotfiles and dotdirs", func(t *testing.T) {
		idx := NewWithOptions(dir, Options{ShowHidden: true})
		if err := idx.Build(); err != nil {
			t.Fatalf("Build: %v", err)
		}
		if idx.Lookup(".gitignore") == nil {
			t.Error(".gitignore should be indexed when ShowHidden=true")
		}
		if idx.Lookup(".hidden_dir") == nil {
			t.Error(".hidden_dir should be indexed when ShowHidden=true")
		}
		if idx.Lookup(filepath.Join(".hidden_dir", "note.md")) == nil {
			t.Error("files inside .hidden_dir should be indexed when ShowHidden=true")
		}
	})

	t.Run("VCS metadata stays hidden even with ShowHidden", func(t *testing.T) {
		idx := NewWithOptions(dir, Options{ShowHidden: true})
		if err := idx.Build(); err != nil {
			t.Fatalf("Build: %v", err)
		}
		if idx.Lookup(".git") != nil {
			t.Error(".git should always be skipped")
		}
		if idx.Lookup(filepath.Join(".git", "config")) != nil {
			t.Error("contents of .git should never be indexed")
		}
	})
}

func TestIgnorePatterns(t *testing.T) {
	dir := setupTestDir(t)
	os.WriteFile(filepath.Join(dir, "temp.log"), []byte("log"), 0o644)

	idx := NewWithOptions(dir, Options{IgnorePatterns: []string{"*.log"}})
	idx.Build()

	if e := idx.Lookup("temp.log"); e != nil {
		t.Error("*.log should be ignored")
	}
}

func TestIsMarkdown(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"README.md", true},
		{"file.markdown", true},
		{"file.mdown", true},
		{"file.MD", true},
		{"file.go", false},
		{"file.txt", false},
	}
	for _, tt := range tests {
		if got := isMarkdown(tt.name); got != tt.want {
			t.Errorf("isMarkdown(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestGitignore(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\nbuild/\n!important.log\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "build"), 0o755)
	os.WriteFile(filepath.Join(dir, "app.log"), []byte("log"), 0o644)
	os.WriteFile(filepath.Join(dir, "important.log"), []byte("keep"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644)
	os.WriteFile(filepath.Join(dir, "build", "out"), []byte("binary"), 0o644)

	idx := New(dir)
	idx.Build()

	if e := idx.Lookup("main.go"); e == nil {
		t.Error("main.go should not be ignored")
	}
	if e := idx.Lookup("app.log"); e != nil {
		t.Error("app.log should be ignored by *.log")
	}
	// Negation: !important.log should un-ignore
	if e := idx.Lookup("important.log"); e == nil {
		t.Error("important.log should be kept (negation)")
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{"*.go", "main.go", true},
		{"*.go", "main.js", false},
		{"**/*.go", "src/main.go", true},
		{"docs/**", "docs/guide.md", true},
		{"docs/**", "docs", true},
	}
	for _, tt := range tests {
		if got := matchGlob(tt.pattern, tt.name); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}

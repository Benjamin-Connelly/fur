package index

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestIndex(t *testing.T) (*Index, string) {
	t.Helper()
	dir := t.TempDir()

	// Create some markdown files
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# Welcome\nThis is the main readme file for the project."), 0o644)
	os.WriteFile(filepath.Join(dir, "guide.md"), []byte("# User Guide\nHow to use the application effectively."), 0o644)
	os.WriteFile(filepath.Join(dir, "api.md"), []byte("# API Reference\nThe REST API supports JSON and XML formats."), 0o644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs", "install.md"), []byte("# Installation\nRun go install to set up the project."), 0o644)

	idx := New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	return idx, dir
}

func TestNewFulltextIndex_MemoryOnly(t *testing.T) {
	ft, err := NewFulltextIndex("")
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	if ft.path != "" {
		t.Errorf("expected empty path for memory-only index, got %q", ft.path)
	}
}

func TestNewFulltextIndex_Persistent(t *testing.T) {
	dir := t.TempDir()
	ft, err := NewFulltextIndex(dir)
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	if ft.path == "" {
		t.Error("expected non-empty path for persistent index")
	}
}

func TestBuildFromAndSearch(t *testing.T) {
	idx, _ := setupTestIndex(t)

	ft, err := NewFulltextIndex("")
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	if err := ft.BuildFrom(idx); err != nil {
		t.Fatalf("BuildFrom: %v", err)
	}

	results, err := ft.Search("readme", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'readme'")
	}

	found := false
	for _, r := range results {
		if r.Path == "readme.md" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected readme.md in results, got %v", results)
	}
}

func TestSearchRelevanceOrder(t *testing.T) {
	idx, _ := setupTestIndex(t)

	ft, err := NewFulltextIndex("")
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	if err := ft.BuildFrom(idx); err != nil {
		t.Fatalf("BuildFrom: %v", err)
	}

	// "guide" should rank guide.md highest (appears in title)
	results, err := ft.Search("guide", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'guide'")
	}
	if results[0].Path != "guide.md" {
		t.Errorf("expected guide.md as top result, got %s", results[0].Path)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	ft, err := NewFulltextIndex("")
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	results, err := ft.Search("", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results for empty query, got %d", len(results))
	}
}

func TestUpdate(t *testing.T) {
	idx, dir := setupTestIndex(t)

	ft, err := NewFulltextIndex("")
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	if err := ft.BuildFrom(idx); err != nil {
		t.Fatalf("BuildFrom: %v", err)
	}

	// Add new content via Update
	newFile := filepath.Join(dir, "changelog.md")
	os.WriteFile(newFile, []byte("# Changelog\nVersion 2.0 includes breaking changes to the zebra module."), 0o644)
	if err := ft.Update(newFile, "changelog.md"); err != nil {
		t.Fatalf("Update: %v", err)
	}

	results, err := ft.Search("zebra", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'zebra' after update")
	}
	if results[0].Path != "changelog.md" {
		t.Errorf("expected changelog.md, got %s", results[0].Path)
	}
}

func TestRemove(t *testing.T) {
	idx, _ := setupTestIndex(t)

	ft, err := NewFulltextIndex("")
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	if err := ft.BuildFrom(idx); err != nil {
		t.Fatalf("BuildFrom: %v", err)
	}

	// Verify it exists first
	results, _ := ft.Search("API Reference", 10)
	if len(results) == 0 {
		t.Fatal("expected api.md in index before removal")
	}

	// Remove and verify gone
	if err := ft.Remove("api.md"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	results, _ = ft.Search("API Reference JSON XML", 10)
	for _, r := range results {
		if r.Path == "api.md" {
			t.Error("api.md should not appear after removal")
		}
	}
}

func TestBuildFulltext_Integration(t *testing.T) {
	idx, _ := setupTestIndex(t)

	if err := idx.BuildFulltext(""); err != nil {
		t.Fatalf("BuildFulltext: %v", err)
	}
	defer idx.CloseFulltext()

	if idx.Fulltext == nil {
		t.Fatal("expected Fulltext to be set")
	}

	results, err := idx.Fulltext.Search("install", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'install'")
	}
}

// TestBuildFulltext_PerRootIsolation is the regression guard for lookit-c24:
// distinct served roots must get distinct on-disk Bleve indexes (no shared
// store), and the legacy shared index from older fur builds is purged. A
// shared store pooled content across every served root and was the cross-root
// disclosure surface.
func TestBuildFulltext_PerRootIsolation(t *testing.T) {
	cache := t.TempDir()

	// A legacy shared index from an older fur, at the old global path.
	legacy := filepath.Join(cache, "index.bleve")
	if err := os.MkdirAll(legacy, 0o700); err != nil {
		t.Fatal(err)
	}

	rootA := t.TempDir()
	os.WriteFile(filepath.Join(rootA, "a.md"), []byte("# A\nalphaunique content here\n"), 0o644)
	rootB := t.TempDir()
	os.WriteFile(filepath.Join(rootB, "b.md"), []byte("# B\nbetaunique content here\n"), 0o644)

	idxA := New(rootA)
	idxA.Build()
	if err := idxA.BuildFulltext(cache); err != nil {
		t.Fatalf("build A: %v", err)
	}
	defer idxA.CloseFulltext()
	idxB := New(rootB)
	idxB.Build()
	if err := idxB.BuildFulltext(cache); err != nil {
		t.Fatalf("build B: %v", err)
	}
	defer idxB.CloseFulltext()

	// Legacy shared index removed.
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("legacy shared index should be removed on build")
	}

	// Two distinct per-root index directories.
	entries, err := os.ReadDir(filepath.Join(cache, "roots"))
	if err != nil {
		t.Fatalf("read roots dir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 per-root index dirs, got %d", len(entries))
	}

	// Storage isolation: each index only holds its own root's content.
	if r, _ := idxA.GetFulltext().Search("betaunique", 10); len(r) != 0 {
		t.Errorf("root A index leaked root B content: %d hits", len(r))
	}
	if r, _ := idxB.GetFulltext().Search("alphaunique", 10); len(r) != 0 {
		t.Errorf("root B index leaked root A content: %d hits", len(r))
	}
	// Sanity: each finds its own.
	if r, _ := idxA.GetFulltext().Search("alphaunique", 10); len(r) == 0 {
		t.Error("root A index missing its own content")
	}
}

func TestSearchHighlights(t *testing.T) {
	idx, _ := setupTestIndex(t)

	ft, err := NewFulltextIndex("")
	if err != nil {
		t.Fatalf("NewFulltextIndex: %v", err)
	}
	defer ft.Close()

	if err := ft.BuildFrom(idx); err != nil {
		t.Fatalf("BuildFrom: %v", err)
	}

	results, err := ft.Search("application", 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected results for 'application'")
	}
	// Snippets should contain highlighted fragments
	if len(results[0].Snippets) == 0 {
		t.Error("expected snippets with highlights")
	}
}

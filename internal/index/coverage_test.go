package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestToDOT builds a link graph (including a broken link) and checks the DOT
// output is well-formed.
func TestToDOT(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n[to b](b.md)\n[broken](missing.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B\n"), 0o644)

	idx := New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	g := NewLinkGraph()
	g.BuildFromIndex(idx)

	dot := g.ToDOT()
	if !strings.HasPrefix(dot, "digraph links {") {
		t.Errorf("DOT output not well-formed:\n%s", dot)
	}
	if !strings.Contains(dot, "->") {
		t.Errorf("DOT output has no edges:\n%s", dot)
	}
}

// TestBuildFromIndex confirms the graph is populated from a built index.
func TestBuildFromIndex(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("# A\n[b](b.md)\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B\nback to [a](a.md)\n"), 0o644)

	idx := New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	g := NewLinkGraph()
	g.BuildFromIndex(idx)

	if fwd := g.ForwardLinks("a.md"); len(fwd) != 1 || fwd[0].Target != "b.md" {
		t.Errorf("a.md forward links = %+v, want [b.md]", fwd)
	}
	if back := g.Backlinks("b.md"); len(back) != 1 {
		t.Errorf("b.md backlinks = %+v, want 1", back)
	}
}

// TestNewWithFsAndOptions covers the alternate constructors and Fs()/GetFulltext
// getters.
func TestNewWithFsAndOptions(t *testing.T) {
	dir := t.TempDir()
	osFs := New(dir).Fs()

	idx := NewWithFs(dir, osFs)
	if idx.Fs() == nil {
		t.Error("NewWithFs: Fs() is nil")
	}
	if idx.GetFulltext() != nil {
		t.Error("GetFulltext should be nil before EnableFulltext")
	}

	idx2 := NewWithFsAndOptions(dir, osFs, Options{ShowHidden: true})
	if idx2.Fs() == nil {
		t.Error("NewWithFsAndOptions: Fs() is nil")
	}
}

// TestAddFile adds entries directly and checks they're indexed/looked up.
func TestAddFile(t *testing.T) {
	dir := t.TempDir()
	idx := New(dir)
	idx.AddFile(filepath.Join(dir, "doc.md"), "doc.md", 42, time.Unix(1700000000, 0))
	idx.AddFile(filepath.Join(dir, "img.png"), "img.png", 10, time.Unix(1700000000, 0))

	if e := idx.Lookup("doc.md"); e == nil || !e.IsMarkdown || e.Size != 42 {
		t.Errorf("Lookup(doc.md) = %+v", e)
	}
	if st := idx.Stats(); st.FileCount != 2 || st.TotalSize != 52 {
		t.Errorf("Stats = %+v, want FileCount=2 TotalSize=52", st)
	}
}

// TestWatcher starts a real fsnotify watcher, writes a file, and confirms the
// debounced onChange callback fires, then closes cleanly (no goroutine leak —
// the package's goleak TestMain enforces that).
func TestWatcher(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "seed.md"), []byte("# seed\n"), 0o644)

	idx := New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	graph := NewLinkGraph()
	graph.BuildFromIndex(idx)

	changed := make(chan string, 8)
	w, err := NewWatcher(idx, graph, func(path string) { changed <- path })
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}
	if err := w.Start(); err != nil {
		w.Close()
		t.Fatalf("Start: %v", err)
	}
	defer w.Close()

	// Trigger a write event.
	if err := os.WriteFile(filepath.Join(dir, "new.md"), []byte("# new\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
		// debounced rebuild ran
	case <-time.After(3 * time.Second):
		t.Fatal("watcher did not fire onChange within 3s")
	}
}

// TestFuzzySearchMarkdownExcludesCode filters to markdown entries only.
func TestFuzzySearchMarkdownExcludesCode(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# r\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# n\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package x\n"), 0o644)

	idx := New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	res := idx.FuzzySearchMarkdown("md")
	for _, e := range res {
		if !e.IsMarkdown {
			t.Errorf("FuzzySearchMarkdown returned non-markdown: %s", e.RelPath)
		}
	}
	// A query matching the .go file must not surface it.
	for _, e := range idx.FuzzySearchMarkdown("main") {
		if e.RelPath == "main.go" {
			t.Error("FuzzySearchMarkdown surfaced main.go")
		}
	}
}

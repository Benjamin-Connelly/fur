package tui

import (
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
)

func TestCycleTheme(t *testing.T) {
	tests := []struct {
		current string
		want    string
	}{
		{"auto", "dark"},
		{"dark", "light"},
		{"light", "auto"},
		{"unknown", "auto"},
		{"", "auto"},
	}
	for _, tt := range tests {
		t.Run(tt.current+"->"+tt.want, func(t *testing.T) {
			m := testModel(t)
			m.cfg.Theme = tt.current
			m.cycleTheme()
			if m.cfg.Theme != tt.want {
				t.Errorf("cycleTheme(%q) = %q, want %q", tt.current, m.cfg.Theme, tt.want)
			}
		})
	}
}

func TestCycleTheme_SetsRenderers(t *testing.T) {
	m := testModel(t)
	m.cfg.Theme = "auto"

	m.cycleTheme()

	if m.mdRenderer == nil {
		t.Error("mdRenderer should be set after cycleTheme")
	}
	if m.codeRenderer == nil {
		t.Error("codeRenderer should be set after cycleTheme")
	}
}

func TestCycleTheme_SetsStatusMessage(t *testing.T) {
	m := testModel(t)
	m.cfg.Theme = "auto"

	m.cycleTheme()

	if m.status.message != "Theme: dark" {
		t.Errorf("status message = %q, want %q", m.status.message, "Theme: dark")
	}
}

func TestCycleTheme_ReRendersPreview(t *testing.T) {
	m := testModel(t)
	m.cfg.Theme = "auto"

	// Load a file that exists in the index
	m.preview.filePath = "README.md"
	_, cmd := m.cycleTheme()

	if cmd == nil {
		t.Error("cycleTheme should return a cmd to re-render when a preview is loaded")
	}
}

func TestCycleTheme_NilCmdWithoutPreview(t *testing.T) {
	m := testModel(t)
	m.cfg.Theme = "auto"
	m.preview.filePath = ""

	_, cmd := m.cycleTheme()

	if cmd != nil {
		t.Error("cycleTheme should return nil cmd when no preview is loaded")
	}
}

func TestCycleTheme_FullCycle(t *testing.T) {
	m := testModel(t)
	m.cfg.Theme = "auto"

	m.cycleTheme()
	m.cycleTheme()
	m.cycleTheme()

	if m.cfg.Theme != "auto" {
		t.Errorf("three cycles should return to auto, got %q", m.cfg.Theme)
	}
}

func TestScrollToFragment_ExactMatch(t *testing.T) {
	m := testModel(t)
	rawSource := "# Hello\n\nSome text.\n\n## World\n\nMore text.\n"
	m.preview.SetContent("test.md", "Hello\n\nSome text.\n\nWorld\n\nMore text.")
	m.preview.height = 40

	m.scrollToFragment("world", rawSource)

	if m.preview.cursorLine != 4 {
		t.Errorf("cursor should be at line 4 (World), got %d", m.preview.cursorLine)
	}
}

func TestScrollToFragment_FirstHeading(t *testing.T) {
	m := testModel(t)
	rawSource := "# Hello\n\nSome text.\n"
	m.preview.SetContent("test.md", "Hello\n\nSome text.")
	m.preview.height = 40

	m.scrollToFragment("hello", rawSource)

	if m.preview.cursorLine != 0 {
		t.Errorf("cursor should be at line 0 (Hello), got %d", m.preview.cursorLine)
	}
}

func TestScrollToFragment_DuplicateHeadings(t *testing.T) {
	m := testModel(t)
	rawSource := "# Intro\n\n## Section\n\nFirst.\n\n## Section\n\nSecond.\n"
	m.preview.SetContent("test.md", "Intro\n\nSection\n\nFirst.\n\nSection\n\nSecond.")
	m.preview.height = 40

	// "section-1" should match the second occurrence
	m.scrollToFragment("section-1", rawSource)

	if m.preview.cursorLine != 6 {
		t.Errorf("cursor should be at line 6 (second Section), got %d", m.preview.cursorLine)
	}
}

func TestScrollToFragment_NoMatch(t *testing.T) {
	m := testModel(t)
	rawSource := "# Hello\n\nSome text.\n"
	m.preview.SetContent("test.md", "Hello\n\nSome text.")
	m.preview.height = 40
	m.preview.cursorLine = 0

	m.scrollToFragment("nonexistent", rawSource)

	// Cursor should stay at 0 since nothing matched
	if m.preview.cursorLine != 0 {
		t.Errorf("cursor should stay at 0 for no match, got %d", m.preview.cursorLine)
	}
}

func TestScrollToFragment_PrefixFallback(t *testing.T) {
	m := testModel(t)
	rawSource := "# Some Heading\n\nBody text.\n"
	// The rendered content won't have exact slug match but has a prefix match
	m.preview.SetContent("test.md", "some heading here\n\nBody text.")
	m.preview.height = 40

	m.scrollToFragment("some-heading", rawSource)

	if m.preview.cursorLine != 0 {
		t.Errorf("prefix fallback should land on line 0, got %d", m.preview.cursorLine)
	}
}

func TestScrollToFragment_CaseInsensitive(t *testing.T) {
	m := testModel(t)
	rawSource := "# UPPER Case\n\nContent.\n"
	m.preview.SetContent("test.md", "UPPER Case\n\nContent.")
	m.preview.height = 40

	m.scrollToFragment("upper-case", rawSource)

	if m.preview.cursorLine != 0 {
		t.Errorf("case-insensitive match should find line 0, got %d", m.preview.cursorLine)
	}
}

func TestBuildPreviewLinks_NoFile(t *testing.T) {
	m := testModel(t)
	m.preview.filePath = ""

	m.buildPreviewLinks()

	if len(m.previewLinks) != 0 {
		t.Error("buildPreviewLinks should produce no links when no file is loaded")
	}
	if m.previewLinkIdx != -1 {
		t.Errorf("previewLinkIdx should be -1, got %d", m.previewLinkIdx)
	}
}

func TestBuildPreviewLinks_NoLinks(t *testing.T) {
	m := testModel(t)
	m.preview.SetContent("README.md", "# Hello\n\nNo links here.")
	m.preview.height = 40

	m.buildPreviewLinks()

	if len(m.previewLinks) != 0 {
		t.Errorf("expected 0 preview links, got %d", len(m.previewLinks))
	}
}

func TestBuildPreviewLinks_FindsLinks(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Theme = "dark"
	dir := t.TempDir()
	cfg.Root = dir

	idx := index.New(dir)
	_ = idx.Build()

	g := index.NewLinkGraph()
	g.SetLinks("doc.md", []index.Link{
		{Source: "doc.md", Target: "other.md", Text: "click here"},
		{Source: "doc.md", Target: "another.md", Text: "see also"},
	})

	m := New(cfg, idx, g, nil)
	m.width = 120
	m.height = 40
	m.recalcLayout()

	m.preview.SetContent("doc.md", "Some intro text\nclick here to learn more\nsee also this page\nfooter")
	m.preview.height = 40

	m.buildPreviewLinks()

	if len(m.previewLinks) != 2 {
		t.Fatalf("expected 2 preview links, got %d", len(m.previewLinks))
	}

	if m.previewLinks[0].target != "other.md" {
		t.Errorf("first link target = %q, want %q", m.previewLinks[0].target, "other.md")
	}
	if m.previewLinks[0].renderedLine != 1 {
		t.Errorf("first link line = %d, want 1", m.previewLinks[0].renderedLine)
	}
	if m.previewLinks[0].text != "click here" {
		t.Errorf("first link text = %q, want %q", m.previewLinks[0].text, "click here")
	}

	if m.previewLinks[1].target != "another.md" {
		t.Errorf("second link target = %q, want %q", m.previewLinks[1].target, "another.md")
	}
	if m.previewLinks[1].renderedLine != 2 {
		t.Errorf("second link line = %d, want 2", m.previewLinks[1].renderedLine)
	}
}

func TestBuildPreviewLinks_WithFragment(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Theme = "dark"
	dir := t.TempDir()
	cfg.Root = dir

	idx := index.New(dir)
	_ = idx.Build()

	g := index.NewLinkGraph()
	g.SetLinks("doc.md", []index.Link{
		{Source: "doc.md", Target: "other.md", Text: "link", Fragment: "section"},
	})

	m := New(cfg, idx, g, nil)
	m.width = 120
	m.height = 40
	m.recalcLayout()

	m.preview.SetContent("doc.md", "text with link inside")
	m.preview.height = 40

	m.buildPreviewLinks()

	if len(m.previewLinks) != 1 {
		t.Fatalf("expected 1 preview link, got %d", len(m.previewLinks))
	}
	if m.previewLinks[0].fragment != "section" {
		t.Errorf("fragment = %q, want %q", m.previewLinks[0].fragment, "section")
	}
}

func TestBuildPreviewLinks_UsesTargetWhenTextEmpty(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Theme = "dark"
	dir := t.TempDir()
	cfg.Root = dir

	idx := index.New(dir)
	_ = idx.Build()

	g := index.NewLinkGraph()
	g.SetLinks("doc.md", []index.Link{
		{Source: "doc.md", Target: "readme.md", Text: ""},
	})

	m := New(cfg, idx, g, nil)
	m.width = 120
	m.height = 40
	m.recalcLayout()

	m.preview.SetContent("doc.md", "check readme.md for details")
	m.preview.height = 40

	m.buildPreviewLinks()

	if len(m.previewLinks) != 1 {
		t.Fatalf("expected 1 link, got %d", len(m.previewLinks))
	}
	if m.previewLinks[0].target != "readme.md" {
		t.Errorf("target = %q, want %q", m.previewLinks[0].target, "readme.md")
	}
}

func TestBuildPreviewLinks_NoDuplicateLines(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Theme = "dark"
	dir := t.TempDir()
	cfg.Root = dir

	idx := index.New(dir)
	_ = idx.Build()

	// Two links whose text both appear on the same line
	g := index.NewLinkGraph()
	g.SetLinks("doc.md", []index.Link{
		{Source: "doc.md", Target: "a.md", Text: "foo"},
		{Source: "doc.md", Target: "b.md", Text: "foo"},
	})

	m := New(cfg, idx, g, nil)
	m.width = 120
	m.height = 40
	m.recalcLayout()

	m.preview.SetContent("doc.md", "foo bar baz")
	m.preview.height = 40

	m.buildPreviewLinks()

	// Only one link should match since usedLines prevents duplicate line mapping
	if len(m.previewLinks) != 1 {
		t.Errorf("expected 1 link (deduped by line), got %d", len(m.previewLinks))
	}
}

func TestBuildPreviewLinks_ResetsState(t *testing.T) {
	m := testModel(t)
	m.previewLinks = []previewLink{{renderedLine: 5, target: "old.md"}}
	m.previewLinkIdx = 2
	m.preview.highlightLine = 3
	m.preview.filePath = ""

	m.buildPreviewLinks()

	if len(m.previewLinks) != 0 {
		t.Error("previewLinks should be cleared")
	}
	if m.previewLinkIdx != -1 {
		t.Errorf("previewLinkIdx should be -1, got %d", m.previewLinkIdx)
	}
	if m.preview.highlightLine != -1 {
		t.Errorf("highlightLine should be -1, got %d", m.preview.highlightLine)
	}
}

func TestFindRenderedLine(t *testing.T) {
	m := testModel(t)
	m.preview.SetContent("test.md", "Introduction\nSection A\nBody\nSection A\nEnd")
	m.preview.height = 40

	tests := []struct {
		text       string
		occurrence int
		want       int
	}{
		{"Introduction", 0, 0},
		{"Section A", 0, 1},
		{"Section A", 1, 3},
		{"nonexistent", 0, -1},
	}
	for _, tt := range tests {
		got := m.findRenderedLine(tt.text, tt.occurrence)
		if got != tt.want {
			t.Errorf("findRenderedLine(%q, %d) = %d, want %d", tt.text, tt.occurrence, got, tt.want)
		}
	}
}

func TestFindRenderedLine_FallsBackToLastMatch(t *testing.T) {
	m := testModel(t)
	m.preview.SetContent("test.md", "Alpha\nBeta\nAlpha")
	m.preview.height = 40

	// Asking for occurrence 5 when only 2 exist should fall back to last
	got := m.findRenderedLine("Alpha", 5)
	if got != 2 {
		t.Errorf("fallback should return last match at line 2, got %d", got)
	}
}


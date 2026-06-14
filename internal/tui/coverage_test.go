package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
)

// linkedModel builds a Model over a 3-file tree (a.md links to b.md and c.md)
// with the link graph populated and the layout sized.
func linkedModel(t *testing.T) (*Model, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"),
		[]byte("# A\n\nsee [b](b.md) and [c](c.md)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("# B\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "c.md"), []byte("# C\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Theme = "dark"
	cfg.Root = dir
	cfg.Git.Enabled = false

	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	g := index.NewLinkGraph()
	g.BuildFromIndex(idx)

	m := New(cfg, idx, g, nil)
	m.width = 120
	m.height = 40
	m.recalcLayout()
	return m, dir
}

// TestSidePanelViews drives every side-panel view path (TOC, backlinks,
// bookmarks, git info) plus the setters and Select().
func TestSidePanelViews(t *testing.T) {
	m := NewSidePanelModel()

	m.SetTOC([]TOCEntry{{Level: 1, Text: "Top", Line: 1}, {Level: 2, Text: "Sub", Line: 5}})
	m.SetBacklinks([]index.Link{{Source: "a.md", Target: "b.md", Text: "to b"}})
	m.AddBookmark(Bookmark{Path: "a.md", Title: "Alpha", Scroll: 3})

	cases := []struct {
		pt   PanelType
		want string
	}{
		{PanelTOC, "Top"},
		{PanelBacklinks, "a.md"},
		{PanelBookmarks, "Alpha"},
	}
	for _, c := range cases {
		m.Toggle(c.pt)
		if m.Type() != c.pt {
			t.Errorf("Type() = %v after Toggle(%v)", m.Type(), c.pt)
		}
		view := m.View()
		if !strings.Contains(view, c.want) {
			t.Errorf("panel %v View() missing %q:\n%s", c.pt, c.want, view)
		}
		if sel := m.Select(); sel == nil {
			t.Errorf("Select() on panel %v returned nil", c.pt)
		}
		m.Toggle(c.pt) // toggle off
	}

	m.Toggle(PanelTOC)
	if !m.Visible() {
		t.Error("Visible() = false after toggling a panel on")
	}
	if m.TypeName() == "" {
		t.Error("TypeName() empty")
	}
}

// TestSidePanelGitInfo covers SetGitInfo for both a real repo (the working
// tree, which is a git checkout under CI) and a non-repo directory.
func TestSidePanelGitInfo(t *testing.T) {
	m := NewSidePanelModel()

	m.SetGitInfo(".", "panels.go") // inside the fur repo
	m.Toggle(PanelGitInfo)
	got := m.View()
	if got == "" {
		t.Error("git-info View() is empty")
	}

	m.SetGitInfo(t.TempDir(), "") // not a repo
	if !strings.Contains(m.View(), "Not a git repository") {
		t.Errorf("expected non-repo message, got:\n%s", m.View())
	}
}

// TestPreviewSearchAndPaging covers PreviewModel search rendering and the
// page-jump distance helper.
func TestPreviewSearchAndPaging(t *testing.T) {
	p := NewPreviewModel()
	p.SetContent("doc.md", "# Heading\n\nalpha beta gamma\nalpha again\n")

	p.height = 40
	if got := p.pageLines(); got != 38 {
		t.Errorf("pageLines() = %d, want 38", got)
	}
	p.height = 1
	if got := p.pageLines(); got != 1 {
		t.Errorf("pageLines() floor = %d, want 1", got)
	}

	p.EnterSearchMode()
	if sv := p.SearchView(); sv == "" {
		// empty input renders as empty cursor; assert it doesn't panic and the
		// mode is active instead.
		if !p.searchMode {
			t.Error("EnterSearchMode did not activate search")
		}
	}
}

// TestLinkNavigation drives the link-follow handlers: showing the overlay for
// a multi-link file, navigating the selection, following, and the command-mode
// links listing.
func TestLinkNavigation(t *testing.T) {
	m, _ := linkedModel(t)
	m.preview.SetContent("a.md", "see b.md and c.md")
	m.preview.filePath = "a.md"

	// a.md has two links → handleFollowLink opens the select overlay.
	if _, _ = m.handleFollowLink(); m.mode != modeLinkSelect {
		t.Errorf("mode = %v after handleFollowLink, want modeLinkSelect", m.mode)
	}

	// Navigate the overlay and select.
	m.handleLinkSelectKey(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := m.handleLinkSelectKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Error("selecting a link produced no command")
	}
	if m.mode != modeNormal {
		t.Errorf("mode = %v after enter, want modeNormal", m.mode)
	}

	// esc path on a fresh overlay.
	m.handleFollowLink()
	m.handleLinkSelectKey(tea.KeyMsg{Type: tea.KeyEsc})

	// handleLinkFollow navigates to a target.
	if _, _ = m.handleLinkFollow("b.md", ""); m.preview.filePath == "" {
		t.Error("handleLinkFollow left preview empty")
	}

	// command-mode links listing.
	m.preview.filePath = "a.md"
	_, cmd = m.handleCommandLinks()
	if cmd == nil {
		t.Fatal("handleCommandLinks returned no command")
	}
	if msg := cmd(); msg == nil {
		t.Error("handleCommandLinks command produced nil msg")
	}
}

// TestHeadingJumpView covers filterHeadingJump (filtered + unfiltered) and the
// headingJumpView rendering.
func TestHeadingJumpView(t *testing.T) {
	m, _ := linkedModel(t)
	m.headingJumpTI = newHeadingInput()
	m.headingJumpItems = []headingJumpEntry{
		{Heading: "Introduction", File: "a.md"},
		{Heading: "Install", File: "b.md"},
		{Heading: "Usage", File: "c.md"},
	}

	if got := m.filterHeadingJump(); len(got) != 3 {
		t.Errorf("unfiltered filter = %d, want 3", len(got))
	}
	m.headingJumpInput = "inst"
	if got := m.filterHeadingJump(); len(got) != 1 || got[0].Heading != "Install" {
		t.Errorf("filtered = %+v, want [Install]", got)
	}

	m.headingJumpInput = ""
	if v := m.headingJumpView(); !strings.Contains(v, "Introduction") {
		t.Errorf("headingJumpView missing entry:\n%s", v)
	}
}

// TestLoadPreview routes a markdown file through loadPreview and executes the
// returned command.
func TestLoadPreview(t *testing.T) {
	m, _ := linkedModel(t)
	entry := m.idx.Lookup("a.md")
	if entry == nil {
		t.Fatal("a.md not in index")
	}
	_, cmd := m.loadPreview(*entry)
	if cmd == nil {
		t.Fatal("loadPreview returned no command")
	}
	if msg := cmd(); msg == nil {
		t.Error("loadPreview command produced nil msg")
	}
}

// TestHandleOpenInput covers the command-palette fuzzy "open" handler.
func TestHandleOpenInput(t *testing.T) {
	m, _ := linkedModel(t)
	p := NewCommandPalette()

	p.input = "open " // no query
	if _, ok := p.HandleOpenInput(m.idx).(StatusMsg); !ok {
		t.Error("empty open query should yield StatusMsg")
	}

	p.input = "open zzz-no-such-file"
	if _, ok := p.HandleOpenInput(m.idx).(StatusMsg); !ok {
		t.Error("no-match open should yield StatusMsg")
	}

	p.input = "open a"
	if _, ok := p.HandleOpenInput(m.idx).(FileSelectedMsg); !ok {
		t.Error("matching open should yield FileSelectedMsg")
	}
}

// TestSetRemoteInfo covers the remote-info setter.
func TestSetRemoteInfo(t *testing.T) {
	m, _ := linkedModel(t)
	m.SetRemoteInfo(&RemoteInfo{Display: "user@example:/docs", State: "Connected"})
	if m.remoteInfo == nil || m.remoteInfo.State != "Connected" {
		t.Error("SetRemoteInfo did not store the info")
	}
}

// TestToggleDir expands and re-collapses a directory node in the file list.
func TestToggleDir(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	os.WriteFile(filepath.Join(dir, "top.md"), []byte("# top\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "sub", "nested.md"), []byte("# n\n"), 0o644)

	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	fl := NewFileListModel(idx)

	// Find the "sub" directory node in the visible list.
	dirIdx := -1
	for i, n := range fl.visible {
		if n.isDir {
			dirIdx = i
			break
		}
	}
	if dirIdx < 0 {
		t.Fatal("no directory node in file list")
	}
	relPath := fl.visible[dirIdx].entry.RelPath
	fl.cursor = dirIdx

	wasCollapsed := fl.collapsed[relPath]
	fl.ToggleDir()
	if fl.collapsed[relPath] == wasCollapsed {
		t.Error("ToggleDir did not flip the collapsed state")
	}
	fl.ToggleDir()
	if fl.collapsed[relPath] != wasCollapsed {
		t.Error("second ToggleDir did not restore the collapsed state")
	}
}

// TestScrollToLink covers both the in-range (highlight + scroll) and
// out-of-range branches of scrollToLink.
func TestScrollToLink(t *testing.T) {
	m, _ := linkedModel(t)
	m.preview.SetContent("a.md", "see b.md and c.md\n"+strings.Repeat("filler\n", 80))
	m.preview.filePath = "a.md"
	m.preview.height = 40
	m.buildPreviewLinks()

	m.previewLinkIdx = 0
	m.scrollToLink()
	if m.preview.highlightLine < 0 {
		t.Error("scrollToLink did not set highlightLine for a valid index")
	}

	m.previewLinkIdx = -1
	m.scrollToLink()
	if m.preview.highlightLine != -1 {
		t.Errorf("out-of-range scrollToLink should clear highlight, got %d", m.preview.highlightLine)
	}
}

// TestIsTextFile covers the extension allowlist.
func TestIsTextFile(t *testing.T) {
	for _, ext := range []string{".go", ".YAML", ".json", ".csv"} {
		if !isTextFile(ext) {
			t.Errorf("isTextFile(%q) = false, want true", ext)
		}
	}
	for _, ext := range []string{".png", ".exe", ".unknownext"} {
		if isTextFile(ext) {
			t.Errorf("isTextFile(%q) = true, want false", ext)
		}
	}
}

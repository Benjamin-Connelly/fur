package tui

import (
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/index"
)

func TestSidePanelModel_New(t *testing.T) {
	m := NewSidePanelModel()
	if m.panelType != PanelNone {
		t.Error("new panel should be PanelNone")
	}
	if m.Visible() {
		t.Error("new panel should not be visible")
	}
}

func TestSidePanelModel_Toggle(t *testing.T) {
	m := NewSidePanelModel()

	m.Toggle(PanelTOC)
	if m.Type() != PanelTOC {
		t.Error("should be PanelTOC")
	}
	if !m.Visible() {
		t.Error("should be visible")
	}
	if m.cursor != 0 {
		t.Error("cursor should reset to 0 on toggle open")
	}

	// Toggle same type closes
	m.Toggle(PanelTOC)
	if m.Type() != PanelNone {
		t.Error("toggling same type should close")
	}
	if m.Visible() {
		t.Error("should not be visible after toggle close")
	}

	// Toggle to different type switches
	m.Toggle(PanelBacklinks)
	if m.Type() != PanelBacklinks {
		t.Error("should switch to PanelBacklinks")
	}
	m.Toggle(PanelBookmarks)
	if m.Type() != PanelBookmarks {
		t.Error("should switch to PanelBookmarks")
	}
}

func TestSidePanelModel_TypeName(t *testing.T) {
	m := NewSidePanelModel()
	tests := []struct {
		pt   PanelType
		want string
	}{
		{PanelTOC, "TOC"},
		{PanelBacklinks, "BACKLINKS"},
		{PanelBookmarks, "BOOKMARKS"},
		{PanelGitInfo, "GIT"},
		{PanelNone, "PANEL"},
	}
	for _, tt := range tests {
		m.panelType = tt.pt
		if got := m.TypeName(); got != tt.want {
			t.Errorf("TypeName(%d) = %q, want %q", tt.pt, got, tt.want)
		}
	}
}

func TestSidePanelModel_TOC(t *testing.T) {
	m := NewSidePanelModel()
	m.Toggle(PanelTOC)

	entries := []TOCEntry{
		{Level: 1, Text: "Title", Line: 1},
		{Level: 2, Text: "Section A", Line: 5},
		{Level: 2, Text: "Section B", Line: 10},
	}
	m.SetTOC(entries)

	if m.itemCount() != 3 {
		t.Errorf("expected 3 TOC items, got %d", m.itemCount())
	}

	// Navigate and select
	m.MoveDown()
	if m.cursor != 1 {
		t.Errorf("cursor should be 1, got %d", m.cursor)
	}

	sel := m.Select()
	if sel == nil {
		t.Fatal("Select should return non-nil for TOC")
	}
	if sel.Line != 5 {
		t.Errorf("expected line 5 for Section A, got %d", sel.Line)
	}
}

func TestSidePanelModel_Backlinks(t *testing.T) {
	m := NewSidePanelModel()
	m.Toggle(PanelBacklinks)

	links := []index.Link{
		{Source: "a.md", Target: "current.md", Text: "see current"},
		{Source: "b.md", Target: "current.md", Text: ""},
	}
	m.SetBacklinks(links)

	if m.itemCount() != 2 {
		t.Errorf("expected 2 backlinks, got %d", m.itemCount())
	}

	sel := m.Select()
	if sel == nil || sel.Path != "a.md" {
		t.Errorf("expected path 'a.md', got %v", sel)
	}
}

func TestSidePanelModel_Bookmarks(t *testing.T) {
	m := NewSidePanelModel()
	m.Toggle(PanelBookmarks)

	m.AddBookmark(Bookmark{Path: "a.md", Title: "First", Scroll: 0})
	m.AddBookmark(Bookmark{Path: "b.md", Title: "Second", Scroll: 10})

	if m.itemCount() != 2 {
		t.Errorf("expected 2 bookmarks, got %d", m.itemCount())
	}

	// Duplicate path updates instead of adding
	m.AddBookmark(Bookmark{Path: "a.md", Title: "Updated First", Scroll: 5})
	if m.itemCount() != 2 {
		t.Error("duplicate path should update, not add")
	}
	if m.bookmarks[0].Title != "Updated First" || m.bookmarks[0].Scroll != 5 {
		t.Error("bookmark should be updated")
	}

	// Select
	m.MoveDown()
	sel := m.Select()
	if sel == nil || sel.Path != "b.md" || sel.Scroll != 10 {
		t.Errorf("expected bookmark b.md@10, got %v", sel)
	}

	// Remove
	m.RemoveBookmark(0)
	if m.itemCount() != 1 {
		t.Error("should have 1 bookmark after remove")
	}
	if m.bookmarks[0].Path != "b.md" {
		t.Error("remaining bookmark should be b.md")
	}
}

func TestSidePanelModel_RemoveBookmark_OutOfBounds(t *testing.T) {
	m := NewSidePanelModel()
	m.Toggle(PanelBookmarks)
	m.AddBookmark(Bookmark{Path: "a.md", Title: "A"})

	// Out of bounds removals should be safe
	m.RemoveBookmark(-1)
	m.RemoveBookmark(5)
	if len(m.bookmarks) != 1 {
		t.Error("out-of-bounds remove should be no-op")
	}
}

func TestSidePanelModel_MoveUpDown(t *testing.T) {
	m := NewSidePanelModel()
	m.Toggle(PanelTOC)
	m.SetTOC([]TOCEntry{
		{Text: "A"}, {Text: "B"}, {Text: "C"},
	})

	// Start at 0
	m.MoveUp()
	if m.cursor != 0 {
		t.Error("MoveUp at 0 should be no-op")
	}

	m.MoveDown()
	m.MoveDown()
	if m.cursor != 2 {
		t.Errorf("expected cursor 2, got %d", m.cursor)
	}

	// Clamp at bottom
	m.MoveDown()
	if m.cursor != 2 {
		t.Error("MoveDown at end should be no-op")
	}
}

func TestSidePanelModel_Select_Empty(t *testing.T) {
	m := NewSidePanelModel()
	m.Toggle(PanelTOC)
	// No TOC entries
	if m.Select() != nil {
		t.Error("Select on empty TOC should return nil")
	}
}

func TestSidePanelModel_Select_None(t *testing.T) {
	m := NewSidePanelModel()
	// PanelNone
	if m.Select() != nil {
		t.Error("Select on PanelNone should return nil")
	}
}

func TestSidePanelModel_ItemCount_AllTypes(t *testing.T) {
	m := NewSidePanelModel()

	m.panelType = PanelTOC
	m.toc = []TOCEntry{{}, {}}
	if m.itemCount() != 2 {
		t.Error("TOC item count wrong")
	}

	m.panelType = PanelBacklinks
	m.backlinks = []index.Link{{}, {}, {}}
	if m.itemCount() != 3 {
		t.Error("backlinks item count wrong")
	}

	m.panelType = PanelBookmarks
	m.bookmarks = []Bookmark{{}}
	if m.itemCount() != 1 {
		t.Error("bookmarks item count wrong")
	}

	m.panelType = PanelGitInfo
	if m.itemCount() != 0 {
		t.Error("git info should have 0 items")
	}

	m.panelType = PanelNone
	if m.itemCount() != 0 {
		t.Error("PanelNone should have 0 items")
	}
}

func TestPanelType_Constants(t *testing.T) {
	// Verify iota ordering
	if PanelNone != 0 {
		t.Errorf("PanelNone should be 0, got %d", PanelNone)
	}
	if PanelTOC != 1 {
		t.Errorf("PanelTOC should be 1, got %d", PanelTOC)
	}
	if PanelBacklinks != 2 {
		t.Errorf("PanelBacklinks should be 2, got %d", PanelBacklinks)
	}
	if PanelGitInfo != 3 {
		t.Errorf("PanelGitInfo should be 3, got %d", PanelGitInfo)
	}
	if PanelBookmarks != 4 {
		t.Errorf("PanelBookmarks should be 4, got %d", PanelBookmarks)
	}
}

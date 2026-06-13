package tui

import "testing"

// TestBacklinksRefreshOnNavigation is the regression guard for lookit-7oq: the
// backlinks side panel must update when the user navigates to a new file, not
// only when 'b' is pressed. The test fixture links docs/guide.md -> README.md,
// so README.md has one backlink (from the guide).
func TestBacklinksRefreshOnNavigation(t *testing.T) {
	m := testModel(t)

	// Open the backlinks panel while no file is loaded — it starts empty.
	m.sidePanel.Toggle(PanelBacklinks)
	if m.sidePanel.Type() != PanelBacklinks {
		t.Fatalf("panel type = %v, want PanelBacklinks", m.sidePanel.Type())
	}
	if len(m.sidePanel.backlinks) != 0 {
		t.Fatalf("precondition: backlinks should start empty, got %d", len(m.sidePanel.backlinks))
	}

	// Navigate to README.md via the central content-load message.
	updated, _ := m.Update(previewWithSourceMsg{
		preview:   PreviewLoadedMsg{Path: "README.md", Content: "# Hello\n"},
		rawSource: "# Hello\n",
	})
	m = updated.(*Model)

	if len(m.sidePanel.backlinks) != 1 {
		t.Fatalf("backlinks not refreshed on navigation: got %d, want 1", len(m.sidePanel.backlinks))
	}
	if got := m.sidePanel.backlinks[0].Source; got != "docs/guide.md" {
		t.Errorf("backlink source = %q, want docs/guide.md", got)
	}

	// Navigating to a file with no backlinks must clear the panel, not retain
	// the previous file's list.
	updated, _ = m.Update(previewWithSourceMsg{
		preview:   PreviewLoadedMsg{Path: "docs/guide.md", Content: "# Guide\n"},
		rawSource: "# Guide\n",
	})
	m = updated.(*Model)
	if len(m.sidePanel.backlinks) != 0 {
		t.Errorf("stale backlinks after navigating to a file with none: got %d, want 0", len(m.sidePanel.backlinks))
	}
}

// TestSetBacklinksResetsCursor guards that replacing the backlinks list resets
// the selection so a stale index can't highlight the wrong row.
func TestSetBacklinksResetsCursor(t *testing.T) {
	var p SidePanelModel
	p.cursor = 5
	p.SetBacklinks(nil)
	if p.cursor != 0 {
		t.Errorf("cursor = %d after SetBacklinks, want 0", p.cursor)
	}
}

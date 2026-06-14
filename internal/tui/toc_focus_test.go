package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestTOCSelectMovesFocusToPreview is the regression guard for lookit-ag7:
// selecting a TOC entry (enter) must hand focus to the document so navigation
// keys continue reading from the heading. Previously focus stayed on the side
// panel, so j/k kept moving the TOC cursor.
func TestTOCSelectMovesFocusToPreview(t *testing.T) {
	m := testModel(t)

	// Load a doc with headings so currentRawSource is set and the TOC has
	// entries to select.
	src := "# Alpha\n\nintro text\n\n## Bravo\n\nmore text\n\n## Charlie\n\ntail\n"
	updated, _ := m.Update(previewWithSourceMsg{
		preview:   PreviewLoadedMsg{Path: "doc.md", Content: src},
		rawSource: src,
	})
	m = updated.(*Model)

	// Open + focus the TOC panel.
	m, _ = sendKey(m, "t")
	if m.sidePanel.Type() != PanelTOC {
		t.Fatalf("expected TOC panel, got %v", m.sidePanel.Type())
	}
	if m.focus != PanelSide {
		t.Fatalf("opening TOC should focus the side panel, got %v", m.focus)
	}

	// Move to the second entry and select it.
	m, _ = sendKey(m, "j")
	m, _ = sendSpecialKey(m, tea.KeyEnter)

	if m.focus != PanelPreview {
		t.Errorf("after selecting a TOC entry, focus = %v, want PanelPreview "+
			"(so navigation keys read the document)", m.focus)
	}
	// The preview cursor should sit at the jumped-to heading line, so j/k
	// continue from there rather than from the top.
	if m.preview.cursorLine == 0 {
		t.Errorf("preview cursor did not move to the selected heading (cursorLine=0)")
	}
}

// TestTOCSelectClampsToDocument guards that a TOC line beyond the rendered
// content does not push the cursor/scroll out of range.
func TestTOCSelectClampsToDocument(t *testing.T) {
	m := testModel(t)
	src := "# Only\n\nshort\n"
	updated, _ := m.Update(previewWithSourceMsg{
		preview:   PreviewLoadedMsg{Path: "doc.md", Content: src},
		rawSource: src,
	})
	m = updated.(*Model)

	m, _ = sendKey(m, "t")
	m, _ = sendSpecialKey(m, tea.KeyEnter)

	if m.focus != PanelPreview {
		t.Errorf("focus = %v, want PanelPreview", m.focus)
	}
	if m.preview.cursorLine < 0 || m.preview.cursorLine >= len(m.preview.lines) {
		t.Errorf("cursorLine %d out of range [0,%d)", m.preview.cursorLine, len(m.preview.lines))
	}
	if m.preview.scroll > m.preview.maxScroll() {
		t.Errorf("scroll %d exceeds maxScroll %d", m.preview.scroll, m.preview.maxScroll())
	}
}

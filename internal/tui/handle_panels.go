package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleSidePanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.sidePanel.MoveUp()
		return m, nil
	case "down", "j":
		m.sidePanel.MoveDown()
		return m, nil
	case "enter":
		sel := m.sidePanel.Select()
		if sel == nil {
			return m, nil
		}
		if sel.Path != "" {
			// Navigate to file (backlinks or bookmarks)
			return m.navigateToPath(sel.Path, sel.Scroll)
		}
		if sel.Line > 0 {
			// Jump the preview to the heading (TOC) and hand focus to the
			// document so navigation keys continue reading from there —
			// previously focus stayed in the TOC panel, so j/k kept moving the
			// TOC cursor instead of scrolling the document.
			m.preview.CursorTo(sel.Line - 1)        // clamps + places the cursor
			m.preview.scroll = m.preview.cursorLine // heading at the top
			if m.preview.scroll > m.preview.maxScroll() {
				m.preview.scroll = m.preview.maxScroll()
			}
			m.focus = PanelPreview
			m.status.SetMode(m.modeString())
			return m, nil
		}
		return m, nil
	case "d":
		// Delete bookmark if in bookmarks panel
		if m.sidePanel.Type() == PanelBookmarks {
			m.sidePanel.RemoveBookmark(m.sidePanel.cursor)
			return m, nil
		}
		return m, nil
	}
	return m, nil
}

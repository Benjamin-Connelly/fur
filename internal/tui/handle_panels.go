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
			// Scroll preview to line (TOC)
			m.preview.scroll = sel.Line - 1
			if m.preview.scroll < 0 {
				m.preview.scroll = 0
			}
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

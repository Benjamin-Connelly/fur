package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleFileListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.fileList.MoveUp()
		return m, nil
	case "down", "j":
		m.fileList.MoveDown()
		return m, nil
	case "enter", "l":
		// If filter is active (frozen results), select from filtered list
		if m.fileList.filter != "" {
			sel := m.fileList.Selected()
			if sel == nil {
				return m, nil
			}
			// Clear filter and open the file
			m.fileList.ClearFilter()
			if sel.IsDir {
				return m, nil
			}
			return m, func() tea.Msg {
				return FileSelectedMsg{Entry: *sel}
			}
		}
		sel := m.fileList.SelectedVisible()
		if sel == nil {
			return m, nil
		}
		if sel.IsDir {
			m.fileList.ToggleDir()
			return m, nil
		}
		return m, func() tea.Msg {
			return FileSelectedMsg{Entry: *sel}
		}
	case "h":
		// Collapse current directory or go to parent
		sel := m.fileList.SelectedVisible()
		if sel != nil && sel.IsDir && !m.fileList.collapsed[sel.RelPath] {
			m.fileList.ToggleDir()
		}
		return m, nil
	case "e":
		return m.openInEditor()
	case "g":
		m.fileList.cursor = 0
		m.fileList.offset = 0
		return m, nil
	case "G":
		max := m.fileList.listLen() - 1
		if max >= 0 {
			m.fileList.cursor = max
		}
		return m, nil
	}
	return m, nil
}

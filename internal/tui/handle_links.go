package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleLinkSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.navigator.CloseLinks()
		return m, nil
	case "up", "k":
		m.navigator.LinkMoveUp()
		return m, nil
	case "down", "j":
		m.navigator.LinkMoveDown()
		return m, nil
	case "enter":
		target, fragment := m.navigator.LinkSelect()
		if target != "" {
			return m, func() tea.Msg {
				return LinkFollowMsg{Target: target, Fragment: fragment}
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) handleFollowLink() (tea.Model, tea.Cmd) {
	if m.preview.filePath == "" {
		return m, nil
	}
	target, fragment := m.navigator.ShowLinks(m.preview.filePath)
	if target != "" {
		// Single link, follow directly
		return m, func() tea.Msg {
			return LinkFollowMsg{Target: target, Fragment: fragment}
		}
	}
	// Either no links (status message) or multiple (overlay shown)
	if !m.navigator.IsShowingLinks() {
		return m, func() tea.Msg {
			return StatusMsg{Text: "No links in current file"}
		}
	}
	return m, nil
}

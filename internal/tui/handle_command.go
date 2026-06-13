package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In text input mode, only use non-character keys for navigation
	// (arrows + ctrl combos). Single characters go to input.
	k := msg.String()
	switch k {
	case "esc":
		m.mode = modeNormal
		m.cmdPalette.Close()
		m.status.SetMode(m.modeString())
		return m, nil
	case "enter":
		// :N — jump to line number (like vim)
		if lineNum, err := strconv.Atoi(strings.TrimSpace(m.cmdPalette.input)); err == nil && lineNum > 0 {
			m.mode = modeNormal
			m.cmdPalette.Close()
			m.status.SetMode(m.modeString())
			target := lineNum - 1 // 0-based scroll
			if target > m.preview.maxScroll() {
				target = m.preview.maxScroll()
			}
			m.preview.scroll = target
			m.focus = PanelPreview
			m.status.SetMessage(fmt.Sprintf("Line %d", lineNum))
			return m, nil
		}
		if strings.HasPrefix(m.cmdPalette.input, "open ") {
			m.mode = modeNormal
			result := m.cmdPalette.HandleOpenInput(m.idx)
			m.status.SetMode(m.modeString())
			if result == nil {
				return m, nil
			}
			return m, func() tea.Msg { return result }
		}
		m.mode = modeNormal
		result := m.cmdPalette.Execute()
		m.status.SetMode(m.modeString())
		if result == nil {
			return m, nil
		}
		return m, func() tea.Msg { return result }
	case "up", "ctrl+p", "ctrl+k":
		m.cmdPalette.MoveUp()
		return m, nil
	case "down", "ctrl+n", "ctrl+j":
		m.cmdPalette.MoveDown()
		return m, nil
	default:
		// Editing (printable runes, backspace, left/right, home/end, ctrl+a/e,
		// ctrl+w word-delete, ctrl+u) goes to the textinput, which re-syncs the
		// input string and refilters the command list.
		cmd := m.cmdPalette.UpdateInput(msg)
		return m, cmd
	}
}

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
		m.cmdPalette.Close()
		m.status.SetMode(m.modeString())
		return m, nil
	case "enter":
		// :N — jump to line number (like vim)
		if lineNum, err := strconv.Atoi(strings.TrimSpace(m.cmdPalette.input)); err == nil && lineNum > 0 {
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
			result := m.cmdPalette.HandleOpenInput(m.idx)
			m.status.SetMode(m.modeString())
			if result == nil {
				return m, nil
			}
			return m, func() tea.Msg { return result }
		}
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
	case "backspace":
		if len(m.cmdPalette.input) > 0 {
			m.cmdPalette.SetInput(m.cmdPalette.input[:len(m.cmdPalette.input)-1])
		}
		return m, nil
	case "ctrl+a":
		// Move cursor to start (clear input) — emacs home
		m.cmdPalette.SetInput("")
		return m, nil
	case "ctrl+u":
		// Kill line — clear input (vim + emacs)
		m.cmdPalette.SetInput("")
		return m, nil
	case "ctrl+w":
		// Delete last word
		input := m.cmdPalette.input
		input = strings.TrimRight(input, " ")
		if i := strings.LastIndex(input, " "); i >= 0 {
			m.cmdPalette.SetInput(input[:i+1])
		} else {
			m.cmdPalette.SetInput("")
		}
		return m, nil
	default:
		if len(k) == 1 {
			m.cmdPalette.SetInput(m.cmdPalette.input + k)
		}
		return m, nil
	}
}

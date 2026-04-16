package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) handleHeadingJumpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.headingJump = false
		m.status.SetMode(m.modeString())
		return m, nil
	case "enter":
		filtered := m.filterHeadingJump()
		if m.headingJumpCur >= 0 && m.headingJumpCur < len(filtered) {
			entry := filtered[m.headingJumpCur]
			m.headingJump = false
			m.status.SetMode(m.modeString())
			m.pendingFragment = slugify(entry.Heading)
			return m.navigateToPath(entry.File, 0)
		}
		m.headingJump = false
		m.status.SetMode(m.modeString())
		return m, nil
	case "up", "ctrl+p", "ctrl+k":
		if m.headingJumpCur > 0 {
			m.headingJumpCur--
		}
		return m, nil
	case "down", "ctrl+n", "ctrl+j":
		filtered := m.filterHeadingJump()
		if m.headingJumpCur < len(filtered)-1 {
			m.headingJumpCur++
		}
		return m, nil
	case "backspace":
		if len(m.headingJumpInput) > 0 {
			m.headingJumpInput = m.headingJumpInput[:len(m.headingJumpInput)-1]
			m.headingJumpCur = 0
		}
		return m, nil
	case "ctrl+u":
		m.headingJumpInput = ""
		m.headingJumpCur = 0
		return m, nil
	default:
		ch := msg.String()
		if len(ch) == 1 {
			m.headingJumpInput += ch
			m.headingJumpCur = 0
		}
		return m, nil
	}
}

func (m *Model) headingJumpView() string {
	var b strings.Builder
	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	b.WriteString(prompt.Render("Jump to heading: ") + m.headingJumpInput)
	b.WriteString("_\n")

	filtered := m.filterHeadingJump()
	maxShow := 10
	if len(filtered) < maxShow {
		maxShow = len(filtered)
	}

	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	for i := 0; i < maxShow; i++ {
		e := filtered[i]
		cursor := "  "
		if i == m.headingJumpCur {
			cursor = "> "
		}
		fmt.Fprintf(&b, "%s%s  %s\n", cursor, e.Heading, dimStyle.Render(e.File))
	}

	if len(filtered) > maxShow {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", len(filtered)-maxShow)))
	}
	if len(filtered) == 0 {
		b.WriteString(dimStyle.Render("  No matching headings"))
	}

	return b.String()
}

func (m *Model) filterHeadingJump() []headingJumpEntry {
	if m.headingJumpInput == "" {
		return m.headingJumpItems
	}
	query := strings.ToLower(m.headingJumpInput)
	var filtered []headingJumpEntry
	for _, e := range m.headingJumpItems {
		if strings.Contains(strings.ToLower(e.Heading), query) ||
			strings.Contains(strings.ToLower(e.File), query) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

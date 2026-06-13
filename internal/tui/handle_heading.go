package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) handleHeadingJumpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeNormal
		m.status.SetMode(m.modeString())
		return m, nil
	case "enter":
		filtered := m.filterHeadingJump()
		if m.headingJumpCur >= 0 && m.headingJumpCur < len(filtered) {
			entry := filtered[m.headingJumpCur]
			m.mode = modeNormal
			m.status.SetMode(m.modeString())
			m.pendingFragment = slugify(entry.Heading)
			return m.navigateToPath(entry.File, 0)
		}
		m.mode = modeNormal
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
	default:
		// Editing (printable runes, backspace, left/right, home/end, ctrl+w,
		// ctrl+u) goes to the textinput; the query mirrors its value and the
		// cursor resets to the top of the refiltered list.
		var cmd tea.Cmd
		m.headingJumpTI, cmd = m.headingJumpTI.Update(msg)
		m.headingJumpInput = m.headingJumpTI.Value()
		m.headingJumpCur = 0
		return m, cmd
	}
}

// newHeadingInput builds the prompt-less textinput backing the heading-jump
// query; the "Jump to heading: " label is rendered by headingJumpView.
func newHeadingInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	return ti
}

func (m *Model) headingJumpView() string {
	var b strings.Builder
	prompt := lipgloss.NewStyle().Foreground(m.ui.Accent).Bold(true)
	b.WriteString(prompt.Render("Jump to heading: ") + m.headingJumpTI.View())
	b.WriteString("\n")

	filtered := m.filterHeadingJump()
	maxShow := 10
	if len(filtered) < maxShow {
		maxShow = len(filtered)
	}

	dimStyle := lipgloss.NewStyle().Foreground(m.ui.Dim)
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

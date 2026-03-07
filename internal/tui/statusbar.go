package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBarModel renders the bottom status bar.
type StatusBarModel struct {
	filePath    string
	message     string
	mode        string
	focus       Panel
	showingHelp bool
	width       int
}

// NewStatusBarModel creates a status bar.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{
		mode: "NORMAL",
	}
}

// SetFile updates the displayed file path.
func (m *StatusBarModel) SetFile(path string) {
	m.filePath = path
}

// SetMessage sets a temporary status message.
func (m *StatusBarModel) SetMessage(msg string) {
	m.message = msg
}

// SetMode sets the current input mode display.
func (m *StatusBarModel) SetMode(mode string) {
	m.mode = mode
}

// contextHints returns panel-specific keybinding hints.
func (m StatusBarModel) contextHints() string {
	if m.showingHelp {
		return "esc:close help  ?:close help  q:quit"
	}
	switch m.focus {
	case PanelPreview:
		return "j/k:scroll  c:copy  e:edit  r:reload  y:link  esc:back"
	case PanelSide:
		return "j/k:scroll  enter:select  d:delete  esc:back"
	default: // PanelFileList
		return "j/k:nav  enter:open  /:filter  e:edit  ?:help  q:quit"
	}
}

// View renders the status bar.
func (m StatusBarModel) View() string {
	modeStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Bold(true).
		Padding(0, 1)

	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252"))

	hintStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("240"))

	modeStr := modeStyle.Render(m.mode)

	var middle string
	if m.filePath != "" {
		middle = " " + m.filePath
	}

	right := ""
	if m.message != "" {
		right = m.message + " "
	} else {
		right = hintStyle.Render(m.contextHints() + " ")
	}

	// Pad middle to fill available width
	modeWidth := lipgloss.Width(modeStr)
	rightWidth := lipgloss.Width(right)
	middleWidth := m.width - modeWidth - rightWidth
	if middleWidth < 0 {
		middleWidth = 0
	}

	paddedMiddle := barStyle.Render(fmt.Sprintf("%-*s", middleWidth, middle))

	return strings.Join([]string{modeStr, paddedMiddle, right}, "")
}

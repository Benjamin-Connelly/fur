package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PreviewModel renders file content in the preview pane.
type PreviewModel struct {
	content       string
	lines         []string
	filePath      string
	scroll        int
	width         int
	height        int
	highlightLine int // -1 = no highlight

	// Source line tracking (for permalink generation)
	sourceLineCount int  // total lines in source file
	isCodeFile      bool // true = rendered lines map 1:1 to source

	// Visual line selection
	visualMode   bool
	visualAnchor int // where selection started (fixed)
	visualStart  int // min(anchor, cursor)
	visualEnd    int // max(anchor, cursor)
	cursorLine   int // current cursor position in visual mode
}

// NewPreviewModel creates a preview pane.
func NewPreviewModel() PreviewModel {
	return PreviewModel{highlightLine: -1}
}

// SetContent updates the preview with rendered content.
func (m *PreviewModel) SetContent(path, content string) {
	m.filePath = path
	m.content = content
	m.lines = strings.Split(content, "\n")
	m.scroll = 0
	m.highlightLine = -1
	m.visualMode = false
	m.cursorLine = 0
}

// SetSourceInfo stores metadata about the source file for line mapping.
func (m *PreviewModel) SetSourceInfo(lineCount int, isCode bool) {
	m.sourceLineCount = lineCount
	m.isCodeFile = isCode
}

// EnterVisualMode starts line selection at the current scroll position.
func (m *PreviewModel) EnterVisualMode() {
	m.visualMode = true
	m.cursorLine = m.scroll
	m.visualAnchor = m.cursorLine
	m.visualStart = m.cursorLine
	m.visualEnd = m.cursorLine
}

// ExitVisualMode clears selection.
func (m *PreviewModel) ExitVisualMode() {
	m.visualMode = false
}

// VisualCursorDown moves the visual cursor down.
func (m *PreviewModel) VisualCursorDown() {
	if m.cursorLine < len(m.lines)-1 {
		m.cursorLine++
	}
	m.updateVisualRange()
	if m.cursorLine >= m.scroll+m.height {
		m.scroll = m.cursorLine - m.height + 1
	}
}

// VisualCursorUp moves the visual cursor up.
func (m *PreviewModel) VisualCursorUp() {
	if m.cursorLine > 0 {
		m.cursorLine--
	}
	m.updateVisualRange()
	if m.cursorLine < m.scroll {
		m.scroll = m.cursorLine
	}
}

func (m *PreviewModel) updateVisualRange() {
	if m.cursorLine < m.visualAnchor {
		m.visualStart = m.cursorLine
		m.visualEnd = m.visualAnchor
	} else {
		m.visualStart = m.visualAnchor
		m.visualEnd = m.cursorLine
	}
}

// SelectedSourceLines returns the 1-based source line range for the visual selection.
// For code files (1:1 mapping), this is exact. For markdown, it's approximate.
func (m *PreviewModel) SelectedSourceLines() (startLine, endLine int) {
	if !m.visualMode {
		line := m.scroll + 1
		return line, line
	}
	return m.visualStart + 1, m.visualEnd + 1
}

// ScrollUp scrolls the preview up.
func (m *PreviewModel) ScrollUp(lines int) {
	m.scroll -= lines
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// ScrollDown scrolls the preview down.
func (m *PreviewModel) ScrollDown(lines int) {
	m.scroll += lines
	maxScroll := m.maxScroll()
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
}

// ScrollToBottom scrolls to the end of content.
func (m *PreviewModel) ScrollToBottom() {
	m.scroll = m.maxScroll()
}

func (m *PreviewModel) maxScroll() int {
	max := len(m.lines) - m.height
	if max < 0 {
		return 0
	}
	return max
}

// gutterWidth returns the character width needed for line numbers.
func (m PreviewModel) gutterWidth() int {
	totalLines := len(m.lines)
	if totalLines < 10 {
		return 2 // " 1 "
	}
	w := 0
	for n := totalLines; n > 0; n /= 10 {
		w++
	}
	return w + 1 // digits + 1 space separator
}

// View renders the preview content with line numbers.
func (m PreviewModel) View() string {
	if m.content == "" {
		placeholder := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Italic(true)
		return placeholder.Render("Select a file to preview")
	}

	if len(m.lines) == 0 {
		return m.content
	}

	end := m.scroll + m.height
	if end > len(m.lines) {
		end = len(m.lines)
	}
	start := m.scroll
	if start >= len(m.lines) {
		start = len(m.lines) - 1
	}
	if start < 0 {
		start = 0
	}

	visible := make([]string, end-start)
	copy(visible, m.lines[start:end])

	// Style definitions
	gutterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("239"))
	gutterSelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true)
	cursorGutterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("226")).Bold(true)
	selStyle := lipgloss.NewStyle().Background(lipgloss.Color("24"))
	linkHlStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("81"))

	gw := m.gutterWidth()
	lineNumFmt := fmt.Sprintf("%%%dd ", gw-1) // right-aligned, trailing space

	var b strings.Builder
	for i, line := range visible {
		lineIdx := start + i
		lineNum := lineIdx + 1 // 1-based

		inSelection := m.visualMode && lineIdx >= m.visualStart && lineIdx <= m.visualEnd
		isCursor := m.visualMode && lineIdx == m.cursorLine

		// Render gutter
		numStr := fmt.Sprintf(lineNumFmt, lineNum)
		if isCursor {
			b.WriteString(cursorGutterStyle.Render(numStr))
		} else if inSelection {
			b.WriteString(gutterSelStyle.Render(numStr))
		} else {
			b.WriteString(gutterStyle.Render(numStr))
		}

		// Render content
		if m.highlightLine == lineIdx {
			b.WriteString(linkHlStyle.Render("▶ " + line))
		} else if inSelection {
			b.WriteString(selStyle.Render(line))
		} else {
			b.WriteString(line)
		}

		if i < len(visible)-1 {
			b.WriteByte('\n')
		}
	}

	result := b.String()

	// Scroll position indicator
	if len(m.lines) > m.height && m.height > 0 {
		pct := 0
		maxS := m.maxScroll()
		if maxS > 0 {
			pct = m.scroll * 100 / maxS
		}
		indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		result += "\n" + indicator.Render(fmt.Sprintf("%s %d%%", strings.Repeat("─", 10), pct))
	}

	return result
}

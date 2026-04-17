package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ansiStripRe strips ANSI escape sequences for plain-text matching in search.
var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

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

	// Cursor tracking (normal + visual mode)
	cursorLine   int  // current cursor position
	scrolloff    int  // margin lines above/below cursor before scrolling
	readingGuide bool // full-row highlight on cursor line

	// Visual line selection
	visualMode   bool
	visualAnchor int // where selection started (fixed)
	visualStart  int // min(anchor, cursor)
	visualEnd    int // max(anchor, cursor)

	// Preview search
	searchMode     bool
	searchQuery    string
	searchMatches  []int // line indices that match
	searchCurrent  int   // index into searchMatches (current match)
	searchHistory  []string
	searchHistIdx  int  // -1 = editing new query, 0+ = browsing history
	searchRegex    bool // true = regex mode, false = substring
	searchRegexErr bool // true if current regex failed to compile
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
	m.searchMode = false
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchCurrent = 0
}

// SetSourceInfo stores metadata about the source file for line mapping.
func (m *PreviewModel) SetSourceInfo(lineCount int, isCode bool) {
	m.sourceLineCount = lineCount
	m.isCodeFile = isCode
}

// CursorDown moves the cursor down one line, scrolling when hitting the scrolloff margin.
func (m *PreviewModel) CursorDown() {
	if m.cursorLine < len(m.lines)-1 {
		m.cursorLine++
	}
	// Scroll when cursor passes the scrolloff margin at the bottom
	bottomMargin := m.scroll + m.height - m.scrolloff - 1
	if m.cursorLine > bottomMargin && m.scroll < m.maxScroll() {
		m.scroll = m.cursorLine - m.height + m.scrolloff + 1
		if m.scroll > m.maxScroll() {
			m.scroll = m.maxScroll()
		}
	}
}

// CursorUp moves the cursor up one line, scrolling when hitting the scrolloff margin.
func (m *PreviewModel) CursorUp() {
	if m.cursorLine > 0 {
		m.cursorLine--
	}
	// Scroll when cursor passes the scrolloff margin at the top
	topMargin := m.scroll + m.scrolloff
	if m.cursorLine < topMargin && m.scroll > 0 {
		m.scroll = m.cursorLine - m.scrolloff
		if m.scroll < 0 {
			m.scroll = 0
		}
	}
}

// CursorTo moves the cursor to a specific line, adjusting scroll.
func (m *PreviewModel) CursorTo(line int) {
	if line < 0 {
		line = 0
	}
	if line >= len(m.lines) {
		line = len(m.lines) - 1
	}
	m.cursorLine = line
	// Ensure cursor is visible
	if m.cursorLine < m.scroll {
		m.scroll = m.cursorLine
	} else if m.cursorLine >= m.scroll+m.height {
		m.scroll = m.cursorLine - m.height + 1
	}
}

// ToggleReadingGuide toggles the full-row cursor highlight.
func (m *PreviewModel) ToggleReadingGuide() {
	m.readingGuide = !m.readingGuide
}

// EnterVisualMode starts line selection at the current cursor position.
func (m *PreviewModel) EnterVisualMode() {
	m.visualMode = true
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

// EnterSearchMode activates search input in the preview pane.
func (m *PreviewModel) EnterSearchMode() {
	m.searchMode = true
	m.searchQuery = ""
	m.searchMatches = nil
	m.searchCurrent = 0
	m.searchHistIdx = -1
}

// ExitSearchMode deactivates search input but keeps match highlights.
func (m *PreviewModel) ExitSearchMode() {
	m.searchMode = false
	if m.searchQuery != "" {
		// Deduplicate: remove if already in history
		filtered := make([]string, 0, len(m.searchHistory))
		for _, h := range m.searchHistory {
			if h != m.searchQuery {
				filtered = append(filtered, h)
			}
		}
		m.searchHistory = append([]string{m.searchQuery}, filtered...)
		if len(m.searchHistory) > 20 {
			m.searchHistory = m.searchHistory[:20]
		}
	}
}

// SearchHistoryUp cycles to the previous search query.
func (m *PreviewModel) SearchHistoryUp() {
	if len(m.searchHistory) == 0 {
		return
	}
	if m.searchHistIdx < len(m.searchHistory)-1 {
		m.searchHistIdx++
		m.searchQuery = m.searchHistory[m.searchHistIdx]
		m.computeMatches()
	}
}

// SearchHistoryDown cycles to the next (more recent) search query.
func (m *PreviewModel) SearchHistoryDown() {
	if m.searchHistIdx <= 0 {
		m.searchHistIdx = -1
		m.searchQuery = ""
		m.computeMatches()
		return
	}
	m.searchHistIdx--
	m.searchQuery = m.searchHistory[m.searchHistIdx]
	m.computeMatches()
}

// SearchInput appends a character to the search query and recomputes matches.
func (m *PreviewModel) SearchInput(r rune) {
	m.searchQuery += string(r)
	m.computeMatches()
}

// SearchBackspace removes the last character from the search query.
func (m *PreviewModel) SearchBackspace() {
	if len(m.searchQuery) > 0 {
		m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		m.computeMatches()
	}
}

// ToggleSearchRegex toggles between substring and regex search modes.
func (m *PreviewModel) ToggleSearchRegex() {
	m.searchRegex = !m.searchRegex
	m.computeMatches()
}

// computeMatches performs case-insensitive search across preview lines.
func (m *PreviewModel) computeMatches() {
	m.searchMatches = nil
	m.searchCurrent = 0
	m.searchRegexErr = false
	if m.searchQuery == "" {
		return
	}

	var re *regexp.Regexp
	if m.searchRegex {
		var err error
		re, err = regexp.Compile("(?i)" + m.searchQuery)
		if err != nil {
			m.searchRegexErr = true
			return
		}
	}

	query := strings.ToLower(m.searchQuery)
	for i, line := range m.lines {
		plain := strings.ToLower(ansiStripRe.ReplaceAllString(line, ""))
		if m.searchRegex {
			if re.MatchString(plain) {
				m.searchMatches = append(m.searchMatches, i)
			}
		} else {
			if strings.Contains(plain, query) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
	}
	if len(m.searchMatches) > 0 {
		m.scrollToMatch()
	}
}

// NextMatch advances to the next search match.
func (m *PreviewModel) NextMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchCurrent++
	if m.searchCurrent >= len(m.searchMatches) {
		m.searchCurrent = 0
	}
	m.scrollToMatch()
}

// PrevMatch goes to the previous search match.
func (m *PreviewModel) PrevMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	m.searchCurrent--
	if m.searchCurrent < 0 {
		m.searchCurrent = len(m.searchMatches) - 1
	}
	m.scrollToMatch()
}

// scrollToMatch scrolls the viewport so the current match is visible.
func (m *PreviewModel) scrollToMatch() {
	if m.searchCurrent < 0 || m.searchCurrent >= len(m.searchMatches) {
		return
	}
	line := m.searchMatches[m.searchCurrent]
	if line < m.scroll || line >= m.scroll+m.height {
		target := line - m.height/3
		if target < 0 {
			target = 0
		}
		m.scroll = target
		max := m.maxScroll()
		if m.scroll > max {
			m.scroll = max
		}
	}
}

// isSearchMatch returns true if the given line index is in searchMatches.
func (m *PreviewModel) isSearchMatch(lineIdx int) bool {
	for _, idx := range m.searchMatches {
		if idx == lineIdx {
			return true
		}
	}
	return false
}

// isCurrentSearchMatch returns true if lineIdx is the currently focused match.
func (m *PreviewModel) isCurrentSearchMatch(lineIdx int) bool {
	if m.searchCurrent < 0 || m.searchCurrent >= len(m.searchMatches) {
		return false
	}
	return m.searchMatches[m.searchCurrent] == lineIdx
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
	readingGuideStyle := lipgloss.NewStyle().Background(lipgloss.Color("238"))
	linkHlStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("81"))
	searchMatchStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("226")).
		Foreground(lipgloss.Color("0"))
	searchCurGutterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226")).
		Bold(true)

	hasSearch := m.searchQuery != "" && len(m.searchMatches) > 0
	queryLower := strings.ToLower(m.searchQuery)

	gw := m.gutterWidth()
	lineNumFmt := fmt.Sprintf("%%%dd ", gw-1) // right-aligned, trailing space

	var b strings.Builder
	for i, line := range visible {
		lineIdx := start + i
		lineNum := lineIdx + 1 // 1-based

		inSelection := m.visualMode && lineIdx >= m.visualStart && lineIdx <= m.visualEnd
		isVisualCursor := m.visualMode && lineIdx == m.cursorLine
		isNormalCursor := !m.visualMode && lineIdx == m.cursorLine
		isMatch := hasSearch && m.isSearchMatch(lineIdx)
		isCurMatch := hasSearch && m.isCurrentSearchMatch(lineIdx)

		// Render gutter — cursor marker in both normal and visual mode
		numStr := fmt.Sprintf(lineNumFmt, lineNum)
		if isNormalCursor && m.readingGuide {
			// Reading guide: gutter gets the guide background too
			guideGutterStyle := cursorGutterStyle.Background(lipgloss.Color("238"))
			b.WriteString(guideGutterStyle.Render(numStr))
		} else if isVisualCursor || isNormalCursor {
			b.WriteString(cursorGutterStyle.Render(numStr))
		} else if isCurMatch {
			b.WriteString(searchCurGutterStyle.Render(numStr))
		} else if inSelection {
			selGutterStyle := gutterSelStyle.Background(lipgloss.Color("24"))
			b.WriteString(selGutterStyle.Render(numStr))
		} else {
			b.WriteString(gutterStyle.Render(numStr))
		}

		// Render content
		if m.highlightLine == lineIdx {
			b.WriteString(linkHlStyle.Render("▶ " + line))
		} else if inSelection {
			plain := ansiStripRe.ReplaceAllString(line, "")
			contentWidth := m.width - gw
			if contentWidth > 0 {
				plain = fmt.Sprintf("%-*s", contentWidth, plain)
			}
			b.WriteString(selStyle.Render(plain))
		} else if isNormalCursor && m.readingGuide {
			// Strip ANSI so background color applies uniformly across the line
			plain := ansiStripRe.ReplaceAllString(line, "")
			contentWidth := m.width - gw
			if contentWidth > 0 {
				plain = fmt.Sprintf("%-*s", contentWidth, plain)
			}
			b.WriteString(readingGuideStyle.Render(plain))
		} else if isMatch {
			b.WriteString(highlightSearchInLine(line, queryLower, searchMatchStyle))
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

// highlightSearchInLine highlights all case-insensitive occurrences of query
// in a line. ANSI sequences are stripped before matching; matched substrings
// are rendered with the given style on the plain text.
func highlightSearchInLine(line, queryLower string, style lipgloss.Style) string {
	plain := ansiStripRe.ReplaceAllString(line, "")
	plainLower := strings.ToLower(plain)

	var positions [][2]int
	searchFrom := 0
	qLen := len(queryLower)
	for {
		idx := strings.Index(plainLower[searchFrom:], queryLower)
		if idx < 0 {
			break
		}
		start := searchFrom + idx
		positions = append(positions, [2]int{start, start + qLen})
		searchFrom = start + qLen
	}

	if len(positions) == 0 {
		return line
	}

	var b strings.Builder
	lastEnd := 0
	for _, pos := range positions {
		if pos[0] > lastEnd {
			b.WriteString(plain[lastEnd:pos[0]])
		}
		b.WriteString(style.Render(plain[pos[0]:pos[1]]))
		lastEnd = pos[1]
	}
	if lastEnd < len(plain) {
		b.WriteString(plain[lastEnd:])
	}
	return b.String()
}

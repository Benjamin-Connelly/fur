package tui

import (
	"fmt"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	gitpkg "github.com/Benjamin-Connelly/fur/internal/git"
)

func (m *Model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.preview.CursorUp()
		return m, nil
	case "down", "j":
		m.preview.CursorDown()
		return m, nil
	case "n":
		m.preview.NextMatch()
		return m, nil
	case "N":
		m.preview.PrevMatch()
		return m, nil
	case "/":
		m.preview.EnterSearchMode()
		m.status.SetMode("SEARCH")
		return m, nil
	case "H":
		m.preview.ToggleReadingGuide()
		return m, nil
	case "pgup", "ctrl+u":
		m.preview.ScrollUp(m.preview.height / 2)
		m.preview.CursorTo(m.preview.scroll + m.preview.scrolloff)
		return m, nil
	case "pgdown", "ctrl+d":
		m.preview.ScrollDown(m.preview.height / 2)
		m.preview.CursorTo(m.preview.scroll + m.preview.height - m.preview.scrolloff - 1)
		return m, nil
	case "u":
		m.preview.ScrollUp(m.preview.height / 2)
		m.preview.CursorTo(m.preview.scroll + m.preview.scrolloff)
		return m, nil
	case "d":
		m.preview.ScrollDown(m.preview.height / 2)
		m.preview.CursorTo(m.preview.scroll + m.preview.height - m.preview.scrolloff - 1)
		return m, nil
	case "home", "g":
		m.preview.CursorTo(0)
		return m, nil
	case "end", "G":
		m.preview.CursorTo(len(m.preview.lines) - 1)
		return m, nil
	case "tab":
		if len(m.previewLinks) > 0 {
			m.previewLinkIdx++
			if m.previewLinkIdx >= len(m.previewLinks) {
				m.previewLinkIdx = 0 // wrap around
			}
			m.scrollToLink()
		}
		return m, nil
	case "shift+tab":
		if len(m.previewLinks) > 0 {
			m.previewLinkIdx--
			if m.previewLinkIdx < 0 {
				m.previewLinkIdx = len(m.previewLinks) - 1 // wrap around
			}
			m.scrollToLink()
		}
		return m, nil
	case "enter":
		if m.previewLinkIdx >= 0 && m.previewLinkIdx < len(m.previewLinks) {
			pl := m.previewLinks[m.previewLinkIdx]
			m.previewLinkIdx = -1
			m.preview.highlightLine = -1
			return m, func() tea.Msg {
				return LinkFollowMsg{Target: pl.target, Fragment: pl.fragment}
			}
		}
		return m, nil
	case "V":
		m.preview.EnterVisualMode()
		m.status.SetMode("VISUAL")
		return m, nil
	case "e":
		return m.openInEditor()
	}
	return m, nil
}

func (m *Model) handlePreviewSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.preview.ExitSearchMode()
		m.status.SetMode(m.modeString())
		return m, nil
	case "backspace":
		m.preview.SearchBackspace()
		return m, nil
	case "ctrl+u":
		m.preview.searchQuery = ""
		m.preview.computeMatches()
		return m, nil
	case "up":
		m.preview.SearchHistoryUp()
		return m, nil
	case "down":
		m.preview.SearchHistoryDown()
		return m, nil
	case "ctrl+r":
		m.preview.ToggleSearchRegex()
		mode := "SEARCH"
		if m.preview.searchRegex {
			mode = "REGEX"
		}
		m.status.SetMode(mode)
		return m, nil
	default:
		ch := msg.String()
		if len(ch) == 1 {
			m.preview.SearchInput(rune(ch[0]))
		}
		return m, nil
	}
}

func (m *Model) handleVisualKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.preview.VisualCursorDown()
		return m, nil
	case "k", "up":
		m.preview.VisualCursorUp()
		return m, nil
	case "y":
		// Copy permalink with selected line range
		startLine, endLine := m.preview.SelectedSourceLines()
		m.preview.ExitVisualMode()
		m.status.SetMode(m.modeString())
		return m, func() tea.Msg {
			repo, err := gitpkg.Open(m.cfg.Root)
			if err != nil {
				return StatusMsg{Text: "Not a git repository"}
			}
			var link string
			if startLine == endLine {
				link, err = repo.CopyPermalink(m.preview.filePath, startLine)
			} else {
				link, err = repo.PermalinkForRange(m.preview.filePath, startLine, endLine)
				if err == nil {
					_ = clipboard.WriteAll(link)
				}
			}
			if err != nil {
				return StatusMsg{Text: "Permalink error: " + err.Error()}
			}
			return StatusMsg{Text: fmt.Sprintf("Copied L%d-%d: %s", startLine, endLine, link)}
		}
	case "esc", "V":
		m.preview.ExitVisualMode()
		m.status.SetMode(m.modeString())
		return m, nil
	case "G":
		// Select to bottom
		m.preview.cursorLine = len(m.preview.lines) - 1
		m.preview.updateVisualRange()
		m.preview.ScrollToBottom()
		return m, nil
	case "g":
		// Select to top
		m.preview.cursorLine = 0
		m.preview.updateVisualRange()
		m.preview.scroll = 0
		return m, nil
	}
	return m, nil
}

func (m *Model) scrollToLink() {
	if m.previewLinkIdx < 0 || m.previewLinkIdx >= len(m.previewLinks) {
		m.preview.highlightLine = -1
		return
	}
	line := m.previewLinks[m.previewLinkIdx].renderedLine
	m.preview.highlightLine = line

	// Scroll so the link is visible, centered if possible
	if line < m.preview.scroll || line >= m.preview.scroll+m.preview.height {
		target := line - m.preview.height/3
		if target < 0 {
			target = 0
		}
		m.preview.scroll = target
		max := m.preview.maxScroll()
		if m.preview.scroll > max {
			m.preview.scroll = max
		}
	}
}

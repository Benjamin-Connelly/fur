package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/afero"

	gitpkg "github.com/Benjamin-Connelly/fur/internal/git"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/plugin"
)

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Command palette intercepts all keys when active
		if m.cmdPalette.IsActive() {
			return m.handleCommandKey(msg)
		}
		// Heading jump intercepts all keys when active
		if m.headingJump {
			return m.handleHeadingJumpKey(msg)
		}
		// Link selection overlay intercepts keys when showing
		if m.navigator.IsShowingLinks() {
			return m.handleLinkSelectKey(msg)
		}
		// Mark register: waiting for a-z after pressing m
		if m.pendingMark {
			m.pendingMark = false
			m.status.SetMode(m.modeString())
			k := msg.String()
			if len(k) == 1 && k[0] >= 'a' && k[0] <= 'z' {
				m.marks[rune(k[0])] = mark{
					File:   m.preview.filePath,
					Cursor: m.preview.cursorLine,
					Scroll: m.preview.scroll,
				}
				m.status.SetMessage("Mark '" + k + "' set")
				return m, m.clearStatusAfter()
			}
			return m, nil
		}
		// Jump to mark: waiting for a-z after pressing '
		if m.pendingJump {
			m.pendingJump = false
			m.status.SetMode(m.modeString())
			k := msg.String()
			if len(k) == 1 && k[0] >= 'a' && k[0] <= 'z' {
				mk, ok := m.marks[rune(k[0])]
				if !ok {
					m.status.SetMessage("Mark '" + k + "' not set")
					return m, m.clearStatusAfter()
				}
				// Navigate to the marked file and position
				if mk.File != m.preview.filePath {
					entry := m.idx.Lookup(mk.File)
					if entry != nil {
						m.preview.scroll = mk.Scroll
						m.preview.cursorLine = mk.Cursor
						return m, func() tea.Msg {
							return FileSelectedMsg{Entry: *entry}
						}
					}
				} else {
					m.preview.scroll = mk.Scroll
					m.preview.cursorLine = mk.Cursor
				}
			}
			return m, nil
		}
		if m.preview.visualMode {
			return m.handleVisualKey(msg)
		}
		if m.preview.searchMode {
			return m.handlePreviewSearchKey(msg)
		}
		if m.fileList.filtering {
			return m.handleFilterKey(msg)
		}
		return m.handleNormalKey(msg)

	case tea.MouseMsg:
		if m.cfg.Mouse {
			switch msg.Type {
			case tea.MouseWheelUp:
				m.preview.ScrollUp(3)
			case tea.MouseWheelDown:
				m.preview.ScrollDown(3)
			}
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		return m, nil

	case FileSelectedMsg:
		m.showingHelp = false
		if m.plugins != nil {
			ctx := &plugin.HookContext{FilePath: msg.Entry.RelPath}
			m.plugins.Run(plugin.HookOnNavigate, ctx)
		}
		if m.recentFiles != nil {
			m.recentFiles.Add(msg.Entry.Path)
			_ = m.recentFiles.Save()
		}
		return m.loadPreview(msg.Entry)

	case PreviewLoadedMsg:
		m.preview.SetContent(msg.Path, msg.Content)
		m.status.SetFile(msg.Path)
		m.status.wordCount = 0
		m.status.readingTime = 0
		m.focus = PanelPreview
		m.status.SetMode(m.modeString())
		m.buildPreviewLinks()
		return m, nil

	case LinkFollowMsg:
		return m.handleLinkFollow(msg.Target, msg.Fragment)

	case commandLinksMsg:
		return m.handleCommandLinks()

	case StatusMsg:
		m.status.SetMessage(msg.Text)
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case previewWithSourceMsg:
		m.preview.SetContent(msg.preview.Path, msg.preview.Content)
		m.status.SetFile(msg.preview.Path)
		m.focus = PanelPreview
		m.status.SetMode(m.modeString())
		m.currentRawSource = msg.rawSource
		// Word count + reading time (avg 200 wpm)
		words := len(strings.Fields(msg.rawSource))
		m.status.wordCount = words
		m.status.readingTime = (words + 199) / 200
		if m.status.readingTime < 1 {
			m.status.readingTime = 1
		}
		m.buildPreviewLinks()
		// Update TOC if panel is open
		if m.sidePanel.Type() == PanelTOC {
			m.sidePanel.SetTOCFromMarkdown(msg.rawSource)
		}
		// Resolve pending anchor fragment
		if m.pendingFragment != "" {
			m.scrollToFragment(m.pendingFragment, msg.rawSource)
			m.pendingFragment = ""
		}
		return m, nil

	case clearStatusMsg:
		m.status.SetMessage("")
		return m, nil
	}

	return m, nil
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "tab":
		if m.sidePanel.Visible() {
			switch m.focus {
			case PanelFileList:
				m.focus = PanelPreview
			case PanelPreview:
				m.focus = PanelSide
			case PanelSide:
				if m.singleFile {
					m.focus = PanelPreview
				} else {
					m.focus = PanelFileList
				}
			}
		} else {
			if m.focus == PanelFileList {
				m.focus = PanelPreview
			} else if !m.singleFile {
				m.focus = PanelFileList
			}
		}
		m.status.SetMode(m.modeString())
		return m, nil

	case "shift+tab":
		// Reverse cycle panels
		if m.sidePanel.Visible() {
			switch m.focus {
			case PanelFileList:
				m.focus = PanelSide
			case PanelPreview:
				if m.singleFile {
					m.focus = PanelSide
				} else {
					m.focus = PanelFileList
				}
			case PanelSide:
				m.focus = PanelPreview
			}
		} else {
			if m.focus == PanelPreview && !m.singleFile {
				m.focus = PanelFileList
			} else if m.focus == PanelFileList {
				m.focus = PanelPreview
			}
		}
		m.status.SetMode(m.modeString())
		return m, nil

	case "esc":
		// Clear search highlights first
		if m.preview.searchQuery != "" {
			m.preview.searchQuery = ""
			m.preview.searchMatches = nil
			m.preview.searchCurrent = 0
			return m, nil
		}
		// Clear link highlight first
		if m.previewLinkIdx >= 0 {
			m.previewLinkIdx = -1
			m.preview.highlightLine = -1
			return m, nil
		}
		// Exit help view first
		if m.showingHelp {
			m.showingHelp = false
			m.preview.SetContent(m.helpPrevPath, m.helpPrevContent)
			m.status.SetFile(m.helpPrevPath)
			return m, nil
		}
		// Clear frozen filter if active
		if m.fileList.filter != "" {
			m.fileList.ClearFilter()
			return m, nil
		}
		// From side panel: close panel and return to preview
		if m.focus == PanelSide {
			m.sidePanel.Toggle(m.sidePanel.Type())
			m.focus = PanelPreview
			m.status.SetMode(m.modeString())
			m.recalcLayout()
			return m, nil
		}
		// From preview: return to file list
		if m.focus == PanelPreview {
			m.focus = PanelFileList
			m.status.SetMode(m.modeString())
		}
		return m, nil

	case "/", "ctrl+k":
		// When preview is focused, / opens preview search instead of file filter
		if msg.String() == "/" && m.focus == PanelPreview {
			m.preview.EnterSearchMode()
			m.status.SetMode("SEARCH")
			return m, nil
		}
		m.focus = PanelFileList
		m.fileList.StartFilter()
		m.status.SetMode("FILTER")
		return m, nil

	case "ctrl+g":
		m.headingJump = true
		m.headingJumpInput = ""
		m.headingJumpItems = m.collectAllHeadings()
		m.headingJumpCur = 0
		m.status.SetMode("HEADING")
		return m, nil

	case "ctrl+t":
		return m.cycleTheme()

	case "?":
		if m.showingHelp {
			// Toggle off — restore previous preview
			m.showingHelp = false
			m.preview.SetContent(m.helpPrevPath, m.helpPrevContent)
			m.status.SetFile(m.helpPrevPath)
			return m, nil
		}
		m.helpPrevPath = m.preview.filePath
		m.helpPrevContent = m.preview.content
		m.showingHelp = true
		content := Help(m.keys)
		m.preview.SetContent("", content)
		m.status.SetFile("Key Bindings")
		m.focus = PanelPreview
		m.status.SetMode(m.modeString())
		return m, nil

	case ":":
		m.cmdPalette.Open()
		m.status.SetMode("COMMAND")
		return m, nil

	case "f":
		return m.handleFollowLink()

	case "t":
		// If already focused on TOC, close it and return to preview
		if m.sidePanel.Type() == PanelTOC && m.focus == PanelSide {
			m.sidePanel.Toggle(PanelTOC)
			m.focus = PanelPreview
			m.status.SetMode(m.modeString())
			m.recalcLayout()
			return m, nil
		}
		// Open TOC (or switch to it) and focus it
		if m.sidePanel.Type() != PanelTOC {
			m.sidePanel.Toggle(PanelTOC)
		}
		if m.currentRawSource != "" {
			m.sidePanel.SetTOCFromMarkdown(m.currentRawSource)
		}
		m.focus = PanelSide
		m.status.SetMode(m.modeString())
		m.recalcLayout()
		return m, nil

	case "b":
		// If already focused on backlinks, close it and return to preview
		if m.sidePanel.Type() == PanelBacklinks && m.focus == PanelSide {
			m.sidePanel.Toggle(PanelBacklinks)
			m.focus = PanelPreview
			m.status.SetMode(m.modeString())
			m.recalcLayout()
			return m, nil
		}
		// Open backlinks (or switch to it) and focus it
		if m.sidePanel.Type() != PanelBacklinks {
			m.sidePanel.Toggle(PanelBacklinks)
		}
		backlinks := m.navigator.BacklinksAt(m.preview.filePath)
		m.sidePanel.SetBacklinks(backlinks)
		m.focus = PanelSide
		m.status.SetMode(m.modeString())
		m.recalcLayout()
		return m, nil

	case "m":
		if m.focus == PanelPreview && m.preview.filePath != "" {
			// Vim-style mark: wait for register key
			m.pendingMark = true
			m.status.SetMode("MARK")
			return m, nil
		}
		// File list: add current file as bookmark
		if m.preview.filePath != "" {
			title := filepath.Base(m.preview.filePath)
			m.sidePanel.AddBookmark(Bookmark{
				Path:   m.preview.filePath,
				Title:  title,
				Scroll: m.preview.scroll,
			})
			return m, func() tea.Msg {
				return StatusMsg{Text: "Bookmarked: " + title}
			}
		}
		return m, nil

	case "'":
		if m.focus == PanelPreview {
			m.pendingJump = true
			m.status.SetMode("JUMP")
			return m, nil
		}
		return m, nil

	case "M":
		// If already focused on bookmarks, close and return to preview
		if m.sidePanel.Type() == PanelBookmarks && m.focus == PanelSide {
			m.sidePanel.Toggle(PanelBookmarks)
			m.focus = PanelPreview
			m.status.SetMode(m.modeString())
			m.recalcLayout()
			return m, nil
		}
		if m.sidePanel.Type() != PanelBookmarks {
			m.sidePanel.Toggle(PanelBookmarks)
		}
		m.focus = PanelSide
		m.status.SetMode(m.modeString())
		m.recalcLayout()
		return m, nil

	case "i":
		// If already focused on git info, close and return to preview
		if m.sidePanel.Type() == PanelGitInfo && m.focus == PanelSide {
			m.sidePanel.Toggle(PanelGitInfo)
			m.focus = PanelPreview
			m.status.SetMode(m.modeString())
			m.recalcLayout()
			return m, nil
		}
		if m.sidePanel.Type() != PanelGitInfo {
			m.sidePanel.Toggle(PanelGitInfo)
		}
		m.sidePanel.SetGitInfo(m.cfg.Root, m.preview.filePath)
		m.focus = PanelSide
		m.status.SetMode(m.modeString())
		m.recalcLayout()
		return m, nil

	case "c":
		if m.preview.filePath == "" {
			return m, nil
		}
		entry := m.idx.Lookup(m.preview.filePath)
		if entry == nil {
			return m, nil
		}
		fs := m.idx.Fs()
		return m, func() tea.Msg {
			data, err := afero.ReadFile(fs, entry.Path)
			if err != nil {
				return StatusMsg{Text: "Read error: " + err.Error()}
			}
			if err := clipboard.WriteAll(string(data)); err != nil {
				return StatusMsg{Text: "Clipboard unavailable: " + err.Error()}
			}
			return StatusMsg{Text: "Copied to clipboard: " + entry.RelPath}
		}

	case "r":
		if m.preview.filePath == "" {
			return m, nil
		}
		entry := m.idx.Lookup(m.preview.filePath)
		if entry == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			return FileSelectedMsg{Entry: *entry}
		}

	case "y":
		if m.preview.filePath == "" {
			return m, nil
		}
		// Use cursor position as line reference
		line := m.preview.cursorLine + 1
		return m, func() tea.Msg {
			repo, err := gitpkg.Open(m.cfg.Root)
			if err != nil {
				return StatusMsg{Text: "Not a git repository"}
			}
			link, err := repo.CopyPermalink(m.preview.filePath, line)
			if err != nil {
				return StatusMsg{Text: "Permalink error: " + err.Error()}
			}
			return StatusMsg{Text: fmt.Sprintf("Copied L%d: %s", line, link)}
		}

	case "backspace":
		entry := m.navigator.Back()
		if entry != nil {
			return m.navigateToPath(entry.Path, entry.Scroll)
		}
		return m, nil

	case "L":
		entry := m.navigator.Forward()
		if entry != nil {
			return m.navigateToPath(entry.Path, entry.Scroll)
		}
		return m, nil

	case "n":
		if m.focus == PanelPreview && len(m.preview.searchMatches) > 0 {
			m.preview.NextMatch()
			return m, nil
		}
		return m, nil

	case "N":
		if m.focus == PanelPreview && len(m.preview.searchMatches) > 0 {
			m.preview.PrevMatch()
			return m, nil
		}
		return m, nil
	}

	// Panel-specific keys
	if m.focus == PanelSide {
		return m.handleSidePanelKey(msg)
	}
	if m.focus == PanelFileList {
		return m.handleFileListKey(msg)
	}
	return m.handlePreviewKey(msg)
}

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

// handlePreviewSearchKey handles keys during preview search mode.
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

// handleVisualKey handles keys during visual line selection mode.
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

// scrollToLink scrolls the preview to bring the current highlighted link into view.
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

func (m *Model) handleSidePanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.sidePanel.MoveUp()
		return m, nil
	case "down", "j":
		m.sidePanel.MoveDown()
		return m, nil
	case "enter":
		sel := m.sidePanel.Select()
		if sel == nil {
			return m, nil
		}
		if sel.Path != "" {
			// Navigate to file (backlinks or bookmarks)
			return m.navigateToPath(sel.Path, sel.Scroll)
		}
		if sel.Line > 0 {
			// Scroll preview to line (TOC)
			m.preview.scroll = sel.Line - 1
			if m.preview.scroll < 0 {
				m.preview.scroll = 0
			}
			return m, nil
		}
		return m, nil
	case "d":
		// Delete bookmark if in bookmarks panel
		if m.sidePanel.Type() == PanelBookmarks {
			m.sidePanel.RemoveBookmark(m.sidePanel.cursor)
			return m, nil
		}
		return m, nil
	}
	return m, nil
}

func (m *Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		// Toggle between filename and content search modes
		if m.searchMode == "filename" {
			m.searchMode = "content"
		} else {
			m.searchMode = "filename"
		}
		m.fileList.searchMode = m.searchMode
		m.applyFilter(m.fileList.filter)
		return m, nil
	case "esc":
		m.fileList.ClearFilter()
		m.searchMode = "filename"
		m.fileList.searchMode = "filename"
		m.status.SetMode("NORMAL")
		return m, nil
	case "enter":
		// Freeze results — stop filtering but keep the filtered list
		m.fileList.filtering = false
		m.focus = PanelFileList
		m.status.SetMode("FILES")
		return m, nil
	case "backspace":
		if len(m.fileList.filter) > 0 {
			m.applyFilter(m.fileList.filter[:len(m.fileList.filter)-1])
		}
		return m, nil
	case "up", "ctrl+p", "ctrl+k":
		m.fileList.MoveUp()
		return m, nil
	case "down", "ctrl+n", "ctrl+j":
		m.fileList.MoveDown()
		return m, nil
	case "ctrl+u":
		m.applyFilter("")
		return m, nil
	case "ctrl+w":
		// Delete last word
		input := m.fileList.filter
		input = strings.TrimRight(input, " ")
		if i := strings.LastIndex(input, " "); i >= 0 {
			m.applyFilter(input[:i+1])
		} else {
			m.applyFilter("")
		}
		return m, nil
	default:
		ch := msg.String()
		// Ignore the `/` that triggered filter mode
		if len(ch) == 1 && ch != "/" {
			m.applyFilter(m.fileList.filter + ch)
		}
		return m, nil
	}
}

// applyFilter updates the file list based on the current search mode.
func (m *Model) applyFilter(query string) {
	if m.searchMode == "content" && m.idx.Fulltext != nil && query != "" {
		results, err := m.idx.Fulltext.Search(query, 50)
		if err == nil {
			entries := make([]index.FileEntry, 0, len(results))
			for _, r := range results {
				if e := m.idx.Lookup(r.Path); e != nil {
					entries = append(entries, *e)
				}
			}
			m.fileList.filter = query
			m.fileList.filtered = entries
			m.fileList.cursor = 0
			m.fileList.offset = 0
			return
		}
	}
	m.fileList.SetFilter(query)
}

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

func (m *Model) navigateToPath(path string, scroll int) (tea.Model, tea.Cmd) {
	entry := m.idx.Lookup(path)
	if entry == nil {
		return m, func() tea.Msg {
			return StatusMsg{Text: "File not found: " + path}
		}
	}

	// Update file list cursor to match
	for i, node := range m.fileList.visible {
		if node.entry.RelPath == path {
			m.fileList.cursor = i
			break
		}
	}

	return m, func() tea.Msg {
		return FileSelectedMsg{Entry: *entry}
	}
}

func (m *Model) openInEditor() (tea.Model, tea.Cmd) {
	// Determine which file to edit
	var filePath string
	if m.focus == PanelFileList {
		sel := m.fileList.SelectedVisible()
		if sel != nil && !sel.IsDir {
			filePath = sel.Path
		}
	} else if m.preview.filePath != "" {
		entry := m.idx.Lookup(m.preview.filePath)
		if entry != nil {
			filePath = entry.Path
		}
	}
	if filePath == "" {
		return m, func() tea.Msg {
			return StatusMsg{Text: "No file selected"}
		}
	}

	// Image files: open with system viewer instead of editor
	ext := filepath.Ext(filePath)
	if IsImageFile(ext) {
		return m.openWithSystem(filePath)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, filePath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return StatusMsg{Text: "Editor error: " + err.Error()}
		}
		// Reload the file after editing
		entry := m.idx.Lookup(m.preview.filePath)
		if entry != nil {
			return FileSelectedMsg{Entry: *entry}
		}
		return StatusMsg{Text: "File edited"}
	})
}

// headingJumpView renders the heading jump picker.
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

// handleHeadingJumpKey handles keys during global heading jump mode.
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

// filterHeadingJump filters heading jump entries by the current query.
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

// clearStatusAfter returns a command that clears the status message after 3 seconds.
func (m *Model) clearStatusAfter() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

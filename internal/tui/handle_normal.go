package tui

import (
	"fmt"
	"path/filepath"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	gitpkg "github.com/Benjamin-Connelly/fur/internal/git"
)

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

package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benjamin-Connelly/fur/internal/plugin"
)

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

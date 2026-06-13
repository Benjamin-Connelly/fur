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
		switch m.mode {
		case modeCommand:
			return m.handleCommandKey(msg)
		case modeHeadingJump:
			return m.handleHeadingJumpKey(msg)
		case modeLinkSelect:
			return m.handleLinkSelectKey(msg)
		case modePendingMark:
			m.mode = modeNormal
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
		case modePendingJump:
			m.mode = modeNormal
			m.status.SetMode(m.modeString())
			k := msg.String()
			if len(k) == 1 && k[0] >= 'a' && k[0] <= 'z' {
				mk, ok := m.marks[rune(k[0])]
				if !ok {
					m.status.SetMessage("Mark '" + k + "' not set")
					return m, m.clearStatusAfter()
				}
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
		case modeVisual:
			return m.handleVisualKey(msg)
		case modeSearch:
			return m.handlePreviewSearchKey(msg)
		case modeFilter:
			return m.handleFilterKey(msg)
		default:
			return m.handleNormalKey(msg)
		}

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
		return m, m.recalcAndReflow()

	case reflowMsg:
		return m.loadPreview(msg.Entry)

	case setThemeMsg:
		return m, m.SetTheme(msg.Name)

	case FileSelectedMsg:
		m.showingHelp = false
		if m.plugins != nil {
			ctx := &plugin.HookContext{FilePath: msg.Entry.RelPath}
			_ = m.plugins.Run(plugin.HookOnNavigate, ctx)
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
		m.restorePreviewScroll()
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
		// Update backlinks if panel is open. Previously backlinks were only
		// populated by the `b` keypress, so navigating to a new file (Enter,
		// follow-link, or clicking a backlink) left the previous file's
		// backlinks displayed.
		if m.sidePanel.Type() == PanelBacklinks {
			m.sidePanel.SetBacklinks(m.navigator.BacklinksAt(m.preview.filePath))
		}
		// Resolve pending anchor fragment
		if m.pendingFragment != "" {
			m.scrollToFragment(m.pendingFragment, msg.rawSource)
			m.pendingFragment = ""
		}
		m.restorePreviewScroll()
		return m, nil

	case clearStatusMsg:
		m.status.SetMessage("")
		return m, nil
	}

	return m, nil
}

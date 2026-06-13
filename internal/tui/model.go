package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/plugin"
	"github.com/Benjamin-Connelly/fur/internal/render"
	"github.com/Benjamin-Connelly/fur/internal/theme"
)

// Panel identifies which panel is currently focused.
type Panel int

const (
	PanelFileList Panel = iota
	PanelPreview
	PanelSide
)

type inputMode int

const (
	modeNormal inputMode = iota
	modeCommand
	modeHeadingJump
	modeLinkSelect
	modePendingMark
	modePendingJump
	modeVisual
	modeSearch
	modeFilter
)

// Message types for inter-component communication.
type FileSelectedMsg struct {
	Entry index.FileEntry
}

// reflowMsg re-renders the open preview at the current width without the
// navigation side effects (recent-files, OnNavigate hook) of FileSelectedMsg.
// Emitted on resize, panel toggles, and theme switches.
type reflowMsg struct {
	Entry index.FileEntry
}

// setThemeMsg switches the active theme. Emitted by the `:theme <name>` palette
// commands so the switch happens on the model rather than in the command closure.
type setThemeMsg struct {
	Name string
}

type PreviewLoadedMsg struct {
	Path    string
	Content string
}

type NavigateMsg struct {
	Path string
}

type StatusMsg struct {
	Text string
}

type clearStatusMsg struct{}

// Model is the root Bubble Tea model for the TUI.
type Model struct {
	cfg      *config.Config
	idx      index.Indexer
	links    *index.LinkGraph
	fileList FileListModel
	preview  PreviewModel
	status   StatusBarModel
	keys     KeyMap

	mdRenderer    *render.MarkdownRenderer
	codeRenderer  *render.CodeRenderer
	imageRenderer *ImageRenderer

	plugins    *plugin.Registry
	navigator  *LinkNavigator
	sidePanel  SidePanelModel
	cmdPalette CommandPalette

	// ui holds the resolved theme chrome colors, propagated to sub-models.
	ui theme.UI

	// Reflow scroll preservation: when set, the next loaded preview restores
	// the scroll/cursor position (clamped) instead of resetting to the top.
	// Used so resizing or switching themes doesn't jump the reader to line 1.
	restoreScroll bool
	pendingScroll int
	pendingCursor int

	mode     inputMode
	focus    Panel
	width    int
	height   int
	quitting bool

	// Track raw markdown source for TOC extraction
	currentRawSource string

	// Help overlay state
	showingHelp     bool
	helpPrevPath    string
	helpPrevContent string

	// Link cursor in preview (Tab/Shift-Tab navigation)
	previewLinks   []previewLink
	previewLinkIdx int // -1 = no link selected

	// Anchor fragment to scroll to after preview loads
	pendingFragment string

	// Global heading jump state
	headingJumpInput string // mirrors headingJumpTI.Value()
	headingJumpTI    textinput.Model
	headingJumpItems []headingJumpEntry
	headingJumpCur   int

	// Recent files persistence
	recentFiles *config.RecentFiles

	// Fulltext search mode toggle: "filename" (default) or "content"
	searchMode string

	// Vim-style marks: m{a-z} sets, '{a-z} jumps
	marks map[rune]mark

	// Remote connection state (nil = local mode)
	remoteInfo *RemoteInfo

	// Single-file mode: hide file list, start in preview
	singleFile bool

	// Pending file to auto-load on Init (set by SelectFile)
	pendingSelect string
}

// RemoteInfo holds remote connection display state for the TUI.
type RemoteInfo struct {
	Display  string // "user@host:/path"
	State    string // "Connected", "Reconnecting", "Disconnected"
	LastSync string // "5s ago", "syncing..."
}

// mark records a position for vim-style marks.
type mark struct {
	File   string
	Cursor int
	Scroll int
}

// headingJumpEntry is a heading from any file in the index.
type headingJumpEntry struct {
	File    string // relative path
	Heading string // heading text
	Line    int    // source line number
}

// previewLink maps a link to its position in the rendered preview.
type previewLink struct {
	renderedLine int    // line number in the rendered content
	target       string // link target path
	fragment     string // anchor fragment (empty if none)
	text         string // link display text
}

// New creates a new root TUI model.
func New(cfg *config.Config, idx index.Indexer, links *index.LinkGraph, plugins *plugin.Registry) *Model {
	km := DefaultKeyMap()
	switch cfg.Keymap {
	case "vim":
		km = VimKeyMap()
	case "emacs":
		km = EmacsKeyMap()
	}

	mdRenderer, _ := render.NewMarkdownRenderer(cfg.Theme, 80)
	mdRenderer.SetFs(idx.Fs())
	codeRenderer := render.NewCodeRenderer(cfg.Theme, true)
	codeRenderer.SetFs(idx.Fs())

	nav := NewLinkNavigator(links)
	panel := NewSidePanelModel()
	palette := NewCommandPalette()
	palette.RegisterCommands(idx, links)

	preview := NewPreviewModel()
	preview.scrolloff = cfg.ScrollOff
	preview.readingGuide = cfg.ReadingGuide

	m := &Model{
		cfg:            cfg,
		idx:            idx,
		links:          links,
		plugins:        plugins,
		fileList:       NewFileListModel(idx),
		preview:        preview,
		status:         NewStatusBarModel(),
		keys:           km,
		mdRenderer:     mdRenderer,
		codeRenderer:   codeRenderer,
		navigator:      nav,
		sidePanel:      panel,
		cmdPalette:     palette,
		focus:          PanelFileList,
		previewLinkIdx: -1,
		recentFiles:    config.LoadRecentFiles(),
		marks:          make(map[rune]mark),
		imageRenderer:  NewImageRenderer(),
		searchMode:     "filename",
	}
	m.applyThemeChrome()
	return m
}

// applyThemeChrome resolves the current theme's UI palette and propagates it to
// the sub-models. Call after the theme changes.
func (m *Model) applyThemeChrome() {
	m.ui = theme.UIFor(m.cfg.Theme)
	m.preview.ui = m.ui
	m.fileList.ui = m.ui
	m.status.ui = m.ui
	m.cmdPalette.ui = m.ui
}

// SetTheme switches the active theme: rebuilds the markdown and code renderers
// at the current preview width, repaints the chrome, and returns a command to
// re-render the open preview (preserving scroll position).
func (m *Model) SetTheme(name string) tea.Cmd {
	if !theme.IsValid(name) {
		return func() tea.Msg { return StatusMsg{Text: "Unknown theme: " + name} }
	}
	m.cfg.Theme = name
	width := m.preview.width - 2
	if width < 1 {
		width = 80
	}
	if r, err := render.NewMarkdownRenderer(name, width); err == nil {
		r.SetFs(m.idx.Fs())
		m.mdRenderer = r
	}
	m.codeRenderer = render.NewCodeRenderer(name, true)
	m.codeRenderer.SetFs(m.idx.Fs())
	m.applyThemeChrome()
	m.status.SetMessage("Theme: " + name)
	if cmd := m.reflowCmd(); cmd != nil {
		return cmd
	}
	return nil
}

// reflowCmd re-renders the open preview at the current width, preserving the
// reader's scroll/cursor position. Returns nil when no file is open.
func (m *Model) reflowCmd() tea.Cmd {
	if m.preview.filePath == "" {
		return nil
	}
	entry := m.idx.Lookup(m.preview.filePath)
	if entry == nil {
		return nil
	}
	m.pendingScroll = m.preview.scroll
	m.pendingCursor = m.preview.cursorLine
	m.restoreScroll = true
	e := *entry
	return func() tea.Msg { return reflowMsg{Entry: e} }
}

// recalcAndReflow recomputes the layout and, if the preview width changed,
// returns a command to re-wrap the open preview at the new width.
func (m *Model) recalcAndReflow() tea.Cmd {
	old := m.preview.width
	m.recalcLayout()
	if m.preview.width != old {
		return m.reflowCmd()
	}
	return nil
}

// restorePreviewScroll re-applies a preserved scroll/cursor after a reflow,
// clamping to the newly rendered content. No-op unless a reflow requested it.
func (m *Model) restorePreviewScroll() {
	if !m.restoreScroll {
		return
	}
	m.restoreScroll = false
	m.preview.CursorTo(m.pendingCursor)
	m.preview.scroll = m.pendingScroll
	if max := m.preview.maxScroll(); m.preview.scroll > max {
		m.preview.scroll = max
	}
	if m.preview.scroll < 0 {
		m.preview.scroll = 0
	}
}

// SelectFile pre-selects a file by relative path on startup and enters
// single-file mode: the file list is hidden by default and focus starts on
// the preview pane (full-width). Press Tab to reveal the file list and
// navigate siblings. The preview auto-loads via Init().
// Must be called before Run().
func (m *Model) SelectFile(relPath string) {
	m.fileList.SelectByPath(relPath)
	m.pendingSelect = relPath
	m.singleFile = true
	m.focus = PanelPreview
}

// SetRemoteInfo updates the remote connection display state.
// Safe to call from any goroutine.
func (m *Model) SetRemoteInfo(info *RemoteInfo) {
	m.remoteInfo = info
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	if m.cfg.Mouse {
		cmds = append(cmds, tea.EnableMouseCellMotion)
	}
	// Auto-load preview for pre-selected file
	if m.pendingSelect != "" {
		entry := m.idx.Lookup(m.pendingSelect)
		if entry != nil {
			cmds = append(cmds, func() tea.Msg {
				return FileSelectedMsg{Entry: *entry}
			})
		}
		m.pendingSelect = ""
	}
	return tea.Batch(cmds...)
}

func (m *Model) recalcLayout() {
	borders := 1
	if m.sidePanel.Visible() {
		borders = 2
	}
	available := m.width - borders
	listWidth := available / 5
	if listWidth < 20 {
		listWidth = 20
	}
	panelWidth := 0
	if m.sidePanel.Visible() {
		panelWidth = (available - listWidth) / 4
		if panelWidth < 25 {
			panelWidth = 25
		}
	}
	previewWidth := available - listWidth - panelWidth

	m.preview.width = previewWidth
	m.preview.height = m.height - 2 // label row + status bar
	m.fileList.height = m.height - 2
	if m.mdRenderer != nil {
		_ = m.mdRenderer.SetWidth(previewWidth - 2)
	}
}

func (m *Model) modeString() string {
	switch m.focus {
	case PanelFileList:
		return "FILES"
	case PanelSide:
		return "PANEL"
	default:
		return "PREVIEW"
	}
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	accentColor := m.ui.Accent
	dimColor := m.ui.Dim
	borderColor := m.ui.Border

	contentHeight := m.height - 1

	// Pane label helper
	paneLabel := func(name string, focused bool, width int) string {
		if focused {
			style := lipgloss.NewStyle().
				Foreground(m.ui.OnAccent).
				Background(accentColor).
				Bold(true).
				Width(width)
			return style.Render(" " + name)
		}
		style := lipgloss.NewStyle().
			Foreground(dimColor).
			Width(width)
		return style.Render(" " + name)
	}

	bodyHeight := contentHeight - 1 // 1 row for label

	// Build each pane as label + body, hard-clipped to exact dimensions.
	buildPane := func(label, content string, width, height int) string {
		body := lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Height(height).
			MaxHeight(height).
			Render(content)
		return lipgloss.JoinVertical(lipgloss.Left, label, body)
	}

	// Narrow mode: <100 cols, show only the focused panel
	narrow := m.width < 80

	var main string
	if narrow {
		w := m.width
		switch m.focus {
		case PanelFileList:
			label := paneLabel("FILES", true, w)
			main = buildPane(label, m.fileList.View(), w, bodyHeight)
		case PanelSide:
			label := paneLabel(m.sidePanel.TypeName(), true, w)
			main = buildPane(label, m.sidePanel.View(), w, bodyHeight)
		default:
			title := m.preview.filePath
			if title == "" {
				title = "PREVIEW"
			}
			label := paneLabel(title, true, w)
			content := m.preview.View()
			if m.navigator.IsShowingLinks() {
				overlay := m.navigator.LinkOverlayView()
				content = overlay + "\n" + strings.Repeat("─", 20) + "\n" + content
			}
			main = buildPane(label, content, w, bodyHeight)
		}
	} else {
		// Normal split-pane layout
		borders := 1
		panelWidth := 0
		showFileList := !m.singleFile || m.focus == PanelFileList

		if m.sidePanel.Visible() {
			borders = 2
		}
		if !showFileList {
			borders--
		}

		available := m.width - borders
		listWidth := 0
		if showFileList {
			listWidth = available / 5
			if listWidth < 20 {
				listWidth = 20
			}
		}

		if m.sidePanel.Visible() {
			panelWidth = (available - listWidth) / 4
			if panelWidth < 25 {
				panelWidth = 25
			}
		}

		previewWidth := available - listWidth - panelWidth

		// Vertical separator
		sepStyle := lipgloss.NewStyle().Foreground(borderColor)
		sep := sepStyle.Render(strings.Repeat("│\n", contentHeight-1) + "│")

		// Preview pane
		previewFocused := m.focus == PanelPreview
		previewTitle := m.preview.filePath
		if previewTitle == "" {
			previewTitle = "PREVIEW"
		}
		previewLabel := paneLabel(previewTitle, previewFocused, previewWidth)
		previewContent := m.preview.View()
		if m.navigator.IsShowingLinks() {
			overlay := m.navigator.LinkOverlayView()
			previewContent = overlay + "\n" + strings.Repeat("─", 20) + "\n" + previewContent
		}

		if showFileList {
			// File list pane
			listFocused := m.focus == PanelFileList || m.fileList.filtering
			listLabel := paneLabel("FILES", listFocused, listWidth)
			left := buildPane(listLabel, m.fileList.View(), listWidth, bodyHeight)

			if m.sidePanel.Visible() {
				right := buildPane(previewLabel, previewContent, previewWidth, bodyHeight)

				sideFocused := m.focus == PanelSide
				sideName := m.sidePanel.TypeName()
				sideLabel := paneLabel(sideName, sideFocused, panelWidth)
				side := buildPane(sideLabel, m.sidePanel.View(), panelWidth, bodyHeight)

				main = lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right, sep, side)
			} else {
				right := buildPane(previewLabel, previewContent, previewWidth, bodyHeight)
				main = lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
			}
		} else {
			// Single-file mode: no file list
			if m.sidePanel.Visible() {
				right := buildPane(previewLabel, previewContent, previewWidth, bodyHeight)

				sideFocused := m.focus == PanelSide
				sideName := m.sidePanel.TypeName()
				sideLabel := paneLabel(sideName, sideFocused, panelWidth)
				side := buildPane(sideLabel, m.sidePanel.View(), panelWidth, bodyHeight)

				main = lipgloss.JoinHorizontal(lipgloss.Top, right, sep, side)
			} else {
				main = buildPane(previewLabel, previewContent, previewWidth, bodyHeight)
			}
		}
	}

	cmdView := m.cmdPalette.View()
	if cmdView != "" {
		return lipgloss.JoinVertical(lipgloss.Left, main, cmdView)
	}

	if m.mode == modeHeadingJump {
		return lipgloss.JoinVertical(lipgloss.Left, main, m.headingJumpView())
	}

	m.status.width = m.width
	m.status.focus = m.focus
	m.status.showingHelp = m.showingHelp
	if m.remoteInfo != nil {
		m.status.remoteDisplay = m.remoteInfo.Display
		m.status.remoteState = m.remoteInfo.State
		m.status.lastSync = m.remoteInfo.LastSync
	}
	m.status.visualMode = m.preview.visualMode
	if m.preview.visualMode {
		s, e := m.preview.SelectedSourceLines()
		m.status.visualRange = fmt.Sprintf("L%d-L%d", s, e)
	} else {
		m.status.visualRange = ""
	}
	m.status.linkActive = m.previewLinkIdx >= 0 && m.previewLinkIdx < len(m.previewLinks)
	if m.status.linkActive {
		m.status.linkText = m.previewLinks[m.previewLinkIdx].text
	} else {
		m.status.linkText = ""
	}
	// Search state for status bar
	m.status.searchMode = m.preview.searchMode
	m.status.searchQuery = m.preview.searchQuery
	if m.preview.searchMode {
		m.status.searchView = m.preview.SearchView()
	} else {
		m.status.searchView = ""
	}
	m.status.searchMatchCount = len(m.preview.searchMatches)
	m.status.searchRegexErr = m.preview.searchRegexErr
	// Filter state for status bar
	m.status.filterActive = !m.fileList.filtering && m.fileList.filter != ""
	m.status.filterQuery = m.fileList.filter
	return lipgloss.JoinVertical(lipgloss.Left, main, m.status.View())
}

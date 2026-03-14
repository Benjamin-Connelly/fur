package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/lookit/internal/config"
	"github.com/Benjamin-Connelly/lookit/internal/index"
	"github.com/Benjamin-Connelly/lookit/internal/plugin"
	"github.com/Benjamin-Connelly/lookit/internal/render"
)

// Panel identifies which panel is currently focused.
type Panel int

const (
	PanelFileList Panel = iota
	PanelPreview
	PanelSide
)

// Message types for inter-component communication.
type FileSelectedMsg struct {
	Entry index.FileEntry
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
	idx      *index.Index
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
	headingJump      bool
	headingJumpInput string
	headingJumpItems []headingJumpEntry
	headingJumpCur   int

	// Recent files persistence
	recentFiles *config.RecentFiles

	// Fulltext search mode toggle: "filename" (default) or "content"
	searchMode string

	// Vim-style marks: m{a-z} sets, '{a-z} jumps
	marks       map[rune]mark
	pendingMark bool // waiting for mark register key
	pendingJump bool // waiting for jump register key

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
func New(cfg *config.Config, idx *index.Index, links *index.LinkGraph, plugins *plugin.Registry) *Model {
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

	return &Model{
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
}

// SelectFile pre-selects a file by relative path on startup.
// In single-file mode (1 non-dir entry), hides the file list and
// focuses the preview pane for full-width content viewing.
// The preview auto-loads via Init().
// Must be called before Run().
func (m *Model) SelectFile(relPath string) {
	m.fileList.SelectByPath(relPath)
	m.pendingSelect = relPath

	// Count non-directory entries to detect single-file mode
	fileCount := 0
	for _, e := range m.idx.Entries() {
		if !e.IsDir {
			fileCount++
		}
	}
	if fileCount <= 1 {
		m.singleFile = true
		m.focus = PanelPreview
	}
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

	accentColor := lipgloss.Color("62")  // bright blue-purple for focused
	dimColor := lipgloss.Color("240")    // gray for unfocused
	borderColor := lipgloss.Color("237") // subtle separator

	contentHeight := m.height - 1

	// Pane label helper
	paneLabel := func(name string, focused bool, width int) string {
		if focused {
			style := lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
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

	if m.headingJump {
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
	m.status.searchMatchCount = len(m.preview.searchMatches)
	m.status.searchRegexErr = m.preview.searchRegexErr
	// Filter state for status bar
	m.status.filterActive = !m.fileList.filtering && m.fileList.filter != ""
	m.status.filterQuery = m.fileList.filter
	return lipgloss.JoinVertical(lipgloss.Left, main, m.status.View())
}

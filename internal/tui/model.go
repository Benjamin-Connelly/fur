package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/lookit/internal/config"
	gitpkg "github.com/Benjamin-Connelly/lookit/internal/git"
	"github.com/Benjamin-Connelly/lookit/internal/index"
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
	marks        map[rune]mark
	pendingMark  bool // waiting for mark register key
	pendingJump  bool // waiting for jump register key

	// Remote connection state (nil = local mode)
	remoteInfo *RemoteInfo
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
func New(cfg *config.Config, idx *index.Index, links *index.LinkGraph) *Model {
	km := DefaultKeyMap()
	switch cfg.Keymap {
	case "vim":
		km = VimKeyMap()
	case "emacs":
		km = EmacsKeyMap()
	}

	mdRenderer, _ := render.NewMarkdownRenderer(cfg.Theme, 80)
	codeRenderer := render.NewCodeRenderer(cfg.Theme, true)

	nav := NewLinkNavigator(links)
	panel := NewSidePanelModel()
	palette := NewCommandPalette()
	palette.RegisterCommands(idx, links)

	preview := NewPreviewModel()
	preview.scrolloff = cfg.ScrollOff
	preview.readingGuide = cfg.ReadingGuide

	return &Model{
		cfg:          cfg,
		idx:          idx,
		links:        links,
		fileList:     NewFileListModel(idx),
		preview:      preview,
		status:       NewStatusBarModel(),
		keys:         km,
		mdRenderer:   mdRenderer,
		codeRenderer: codeRenderer,
		navigator:      nav,
		sidePanel:      panel,
		cmdPalette:     palette,
		focus:          PanelFileList,
		previewLinkIdx:  -1,
		recentFiles:     config.LoadRecentFiles(),
		marks:           make(map[rune]mark),
		imageRenderer:   NewImageRenderer(),
		searchMode:      "filename",
	}
}

// SetRemoteInfo updates the remote connection display state.
// Safe to call from any goroutine.
func (m *Model) SetRemoteInfo(info *RemoteInfo) {
	m.remoteInfo = info
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	if m.cfg.Mouse {
		return tea.EnableMouseCellMotion
	}
	return nil
}

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
		// In preview with links: Tab cycles links instead of switching panels
		if m.focus == PanelPreview && len(m.previewLinks) > 0 {
			return m.handlePreviewKey(msg)
		}
		if m.sidePanel.Visible() {
			// Cycle: FileList -> Preview -> Side -> FileList
			switch m.focus {
			case PanelFileList:
				m.focus = PanelPreview
			case PanelPreview:
				m.focus = PanelSide
			case PanelSide:
				m.focus = PanelFileList
			}
		} else {
			if m.focus == PanelFileList {
				m.focus = PanelPreview
			} else {
				m.focus = PanelFileList
			}
		}
		m.status.SetMode(m.modeString())
		return m, nil

	case "shift+tab":
		// In preview with links: Shift-Tab cycles links backward
		if m.focus == PanelPreview && len(m.previewLinks) > 0 {
			return m.handlePreviewKey(msg)
		}
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
		return m, func() tea.Msg {
			data, err := os.ReadFile(entry.Path)
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

func (m *Model) handleLinkFollow(target, fragment string) (tea.Model, tea.Cmd) {
	// Save current position in history, then push the target so that
	// Back returns the source and Forward from the source reaches the target.
	if m.preview.filePath != "" {
		m.navigator.Navigate(m.preview.filePath, m.preview.scroll)
	}
	m.navigator.Navigate(target, 0)

	// Same-file fragment link: scroll in place without re-rendering
	if fragment != "" && target == m.preview.filePath && m.currentRawSource != "" {
		m.scrollToFragment(fragment, m.currentRawSource)
		return m, nil
	}

	if fragment != "" {
		m.pendingFragment = fragment
	}
	return m.navigateToPath(target, 0)
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

func (m *Model) handleCommandLinks() (tea.Model, tea.Cmd) {
	if m.preview.filePath == "" {
		return m, func() tea.Msg {
			return StatusMsg{Text: "No file open"}
		}
	}
	links := m.navigator.LinksAt(m.preview.filePath)
	if len(links) == 0 {
		return m, func() tea.Msg {
			return StatusMsg{Text: "No links in current file"}
		}
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Links in %s\n", m.preview.filePath))
	b.WriteString(strings.Repeat("=", 40) + "\n\n")
	for _, link := range links {
		status := " "
		if link.Broken {
			status = "!"
		}
		b.WriteString(fmt.Sprintf("  [%s] %s -> %s", status, link.Text, link.Target))
		b.WriteString("\n")
	}
	content := b.String()
	return m, func() tea.Msg {
		return PreviewLoadedMsg{Path: "Links: " + m.preview.filePath, Content: content}
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

// openWithSystem opens a file using the platform's default application.
func (m *Model) openWithSystem(filePath string) (tea.Model, tea.Cmd) {
	opener := "xdg-open" // Linux
	if _, err := exec.LookPath("open"); err == nil {
		opener = "open" // macOS
	}

	c := exec.Command(opener, filePath)
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return StatusMsg{Text: "Open error: " + err.Error()}
		}
		return StatusMsg{Text: "Opened in system viewer"}
	})
}

func (m *Model) loadPreview(entry index.FileEntry) (tea.Model, tea.Cmd) {
	// Capture renderers for closure (safe since they're pointers)
	mdRenderer := m.mdRenderer
	codeRenderer := m.codeRenderer

	imgRenderer := m.imageRenderer

	return m, func() tea.Msg {
		if entry.IsDir {
			return PreviewLoadedMsg{
				Path:    entry.RelPath,
				Content: "[directory]",
			}
		}

		ext := filepath.Ext(entry.RelPath)

		// Image files: text-based info card (terminal image protocols are
		// incompatible with Bubble Tea's alt-screen rendering)
		if IsImageFile(ext) {
			return PreviewLoadedMsg{
				Path:    entry.RelPath,
				Content: imgRenderer.Render(entry.Path),
			}
		}

		data, err := os.ReadFile(entry.Path)
		if err != nil {
			return PreviewLoadedMsg{
				Path:    entry.RelPath,
				Content: "Error: " + err.Error(),
			}
		}

		content := string(data)
		ext = strings.ToLower(ext)

		if entry.IsMarkdown {
			rawSource := content
			var fmCard string
			if fm, body, ok := extractYAMLFrontmatter(content); ok {
				fmCard = renderFrontmatterCard(fm)
				content = body
			}
			if mdRenderer != nil {
				rendered, renderErr := mdRenderer.Render(content)
				if renderErr == nil {
					return previewWithSourceMsg{
						preview: PreviewLoadedMsg{
							Path:    entry.RelPath,
							Content: fmCard + rendered,
						},
						rawSource: rawSource,
					}
				}
			}
		} else if ext == ".json" {
			if formatted, ok := formatJSON(content); ok {
				highlighted, hlErr := codeRenderer.Highlight("data.json", formatted)
				if hlErr == nil {
					content = highlighted
				} else {
					content = formatted
				}
			}
		} else if ext == ".csv" || ext == ".tsv" {
			delim := ','
			if ext == ".tsv" {
				delim = '\t'
			}
			if table, ok := formatCSV(content, delim); ok {
				if mdRenderer != nil {
					rendered, renderErr := mdRenderer.Render(table)
					if renderErr == nil {
						content = rendered
					} else {
						content = table
					}
				} else {
					content = table
				}
			}
		} else {
			if isTextFile(ext) {
				highlighted, hlErr := codeRenderer.Highlight(filepath.Base(entry.RelPath), content)
				if hlErr == nil {
					content = highlighted
				}
			}
		}

		return PreviewLoadedMsg{
			Path:    entry.RelPath,
			Content: content,
		}
	}
}

// previewWithSourceMsg carries both rendered preview and raw markdown source.
type previewWithSourceMsg struct {
	preview   PreviewLoadedMsg
	rawSource string
}

// ansiRe strips ANSI escape sequences for plain-text search.
// Covers SGR, CSI, OSC, and hyperlink sequences that Glamour/lipgloss emit.
var ansiRe = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\]8;[^;]*;[^\x1b]*\x1b\\|\][^\x07]*\x07)`)

// stripANSI removes all ANSI escape sequences from s.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// buildPreviewLinks finds link positions in the rendered preview content.
func (m *Model) buildPreviewLinks() {
	m.previewLinks = nil
	m.previewLinkIdx = -1
	m.preview.highlightLine = -1

	if m.preview.filePath == "" {
		return
	}

	links := m.navigator.LinksAt(m.preview.filePath)
	if len(links) == 0 {
		return
	}

	// Search rendered lines for each link's text
	renderedLines := m.preview.lines
	usedLines := make(map[int]bool) // avoid mapping two links to same line

	for _, link := range links {
		searchText := strings.ToLower(link.Text)
		if searchText == "" {
			searchText = strings.ToLower(link.Target)
		}

		for i, line := range renderedLines {
			if usedLines[i] {
				continue
			}
			plain := strings.ToLower(stripANSI(line))
			if strings.Contains(plain, searchText) {
				m.previewLinks = append(m.previewLinks, previewLink{
					renderedLine: i,
					target:       link.Target,
					fragment:     link.Fragment,
					text:         link.Text,
				})
				usedLines[i] = true
				break
			}
		}
	}
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

		if m.sidePanel.Visible() {
			borders = 2
		}

		available := m.width - borders
		listWidth := available / 5
		if listWidth < 20 {
			listWidth = 20
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

		// File list pane
		listFocused := m.focus == PanelFileList || m.fileList.filtering
		listLabel := paneLabel("FILES", listFocused, listWidth)
		left := buildPane(listLabel, m.fileList.View(), listWidth, bodyHeight)

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
	return lipgloss.JoinVertical(lipgloss.Left, main, m.status.View())
}

func isTextFile(ext string) bool {
	ext = strings.ToLower(ext)
	textExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true, ".rb": true,
		".rs": true, ".c": true, ".h": true, ".cpp": true, ".java": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".yaml": true, ".yml": true, ".toml": true, ".json": true,
		".xml": true, ".html": true, ".css": true, ".scss": true,
		".sql": true, ".lua": true, ".vim": true, ".el": true,
		".txt": true, ".cfg": true, ".ini": true, ".conf": true,
		".mk": true, ".cmake": true, ".dockerfile": true,
		".gitignore": true, ".env": true, ".mod": true, ".sum": true,
		".csv": true, ".tsv": true,
	}
	return textExts[ext]
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
		b.WriteString(fmt.Sprintf("%s%s  %s\n", cursor, e.Heading, dimStyle.Render(e.File)))
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

// slugify converts heading text to a URL-compatible anchor slug.
// Delegates to render.Slugify for consistency across TUI and index.
func slugify(s string) string {
	return render.Slugify(s)
}

// scrollToFragment finds a heading matching the fragment slug and scrolls to it.
// Uses GitHub-style duplicate disambiguation: "heading", "heading-1", "heading-2".
func (m *Model) scrollToFragment(fragment, rawSource string) {
	headings := render.ExtractHeadings(rawSource)
	slug := strings.ToLower(fragment)

	// Build slug -> occurrence index to disambiguate duplicates
	counts := make(map[string]int)
	type match struct {
		text       string
		occurrence int // 0-based occurrence of this heading text
	}
	var candidates []match

	for _, h := range headings {
		base := slugify(h.Text)
		n := counts[base]
		counts[base]++

		effective := base
		if n > 0 {
			effective = base + "-" + strconv.Itoa(n)
		}
		if effective == slug {
			candidates = append(candidates, match{text: h.Text, occurrence: n})
		}
	}

	// Try exact slug match first
	for _, c := range candidates {
		target := m.findRenderedLine(c.text, c.occurrence)
		if target >= 0 {
			m.preview.CursorTo(target)
			return
		}
	}

	// Fallback: case-insensitive prefix match against stripped line content
	for i, line := range m.preview.lines {
		plain := strings.ToLower(strings.TrimSpace(stripANSI(line)))
		if plain != "" && strings.HasPrefix(plain, slug) {
			m.preview.CursorTo(i)
			return
		}
	}
}

// findRenderedLine searches the preview lines for the nth occurrence of
// heading text (0-based). Strips ANSI before matching.
func (m *Model) findRenderedLine(headingText string, occurrence int) int {
	lower := strings.ToLower(headingText)
	seen := 0
	for i, line := range m.preview.lines {
		plain := strings.ToLower(stripANSI(line))
		if strings.Contains(plain, lower) {
			if seen == occurrence {
				return i
			}
			seen++
		}
	}
	// If we didn't find the nth occurrence, try returning the last match
	// (handles cases where Glamour merges lines differently)
	if occurrence > 0 && seen > 0 {
		return m.findRenderedLine(headingText, seen-1)
	}
	return -1
}

// collectAllHeadings gathers headings from every markdown file in the index.
func (m *Model) collectAllHeadings() []headingJumpEntry {
	mdFiles := m.idx.MarkdownFiles()
	var entries []headingJumpEntry
	for _, f := range mdFiles {
		data, err := os.ReadFile(f.Path)
		if err != nil {
			continue
		}
		headings := render.ExtractHeadings(string(data))
		for _, h := range headings {
			entries = append(entries, headingJumpEntry{
				File:    f.RelPath,
				Heading: h.Text,
				Line:    h.Line,
			})
		}
	}
	return entries
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

var themeOrder = []string{"auto", "dark", "light"}

// cycleTheme rotates through auto → dark → light and re-renders.
func (m *Model) cycleTheme() (*Model, tea.Cmd) {
	current := m.cfg.Theme
	next := "auto"
	for i, t := range themeOrder {
		if t == current {
			next = themeOrder[(i+1)%len(themeOrder)]
			break
		}
	}
	m.cfg.Theme = next
	m.mdRenderer, _ = render.NewMarkdownRenderer(next, 80)
	m.codeRenderer = render.NewCodeRenderer(next, true)
	m.status.SetMessage("Theme: " + next)

	// Re-render current preview if one is loaded
	if m.preview.filePath != "" {
		entry := m.idx.Lookup(m.preview.filePath)
		if entry != nil {
			return m, func() tea.Msg {
				return FileSelectedMsg{Entry: *entry}
			}
		}
	}
	return m, nil
}

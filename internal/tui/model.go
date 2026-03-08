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

	mdRenderer   *render.MarkdownRenderer
	codeRenderer *render.CodeRenderer

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
}

// previewLink maps a link to its position in the rendered preview.
type previewLink struct {
	renderedLine int    // line number in the rendered content
	target       string // link target path
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

	return &Model{
		cfg:          cfg,
		idx:          idx,
		links:        links,
		fileList:     NewFileListModel(idx),
		preview:      NewPreviewModel(),
		status:       NewStatusBarModel(),
		keys:         km,
		mdRenderer:   mdRenderer,
		codeRenderer: codeRenderer,
		navigator:      nav,
		sidePanel:      panel,
		cmdPalette:     palette,
		focus:          PanelFileList,
		previewLinkIdx: -1,
	}
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
		// Link selection overlay intercepts keys when showing
		if m.navigator.IsShowingLinks() {
			return m.handleLinkSelectKey(msg)
		}
		if m.preview.visualMode {
			return m.handleVisualKey(msg)
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
		return m.loadPreview(msg.Entry)

	case PreviewLoadedMsg:
		m.preview.SetContent(msg.Path, msg.Content)
		m.status.SetFile(msg.Path)
		m.buildPreviewLinks()
		return m, nil

	case LinkFollowMsg:
		return m.handleLinkFollow(msg.Target)

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
		m.currentRawSource = msg.rawSource
		m.buildPreviewLinks()
		// Update TOC if panel is open
		if m.sidePanel.Type() == PanelTOC {
			m.sidePanel.SetTOCFromMarkdown(msg.rawSource)
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
		m.focus = PanelFileList
		m.fileList.StartFilter()
		m.status.SetMode("FILTER")
		return m, nil

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
		// Add current file as bookmark
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
		// Use current scroll position as line reference
		line := m.preview.scroll + 1
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
		m.preview.ScrollUp(1)
		return m, nil
	case "down", "j":
		m.preview.ScrollDown(1)
		return m, nil
	case "pgup", "ctrl+u":
		m.preview.ScrollUp(m.preview.height / 2)
		return m, nil
	case "pgdown", "ctrl+d":
		m.preview.ScrollDown(m.preview.height / 2)
		return m, nil
	case "u":
		m.preview.ScrollUp(m.preview.height / 2)
		return m, nil
	case "d":
		m.preview.ScrollDown(m.preview.height / 2)
		return m, nil
	case "home", "g":
		m.preview.scroll = 0
		return m, nil
	case "end", "G":
		m.preview.ScrollToBottom()
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
			target := m.previewLinks[m.previewLinkIdx].target
			m.previewLinkIdx = -1
			m.preview.highlightLine = -1
			return m, func() tea.Msg {
				return LinkFollowMsg{Target: target}
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
	case "esc":
		m.fileList.ClearFilter()
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
			m.fileList.SetFilter(m.fileList.filter[:len(m.fileList.filter)-1])
		}
		return m, nil
	case "up", "ctrl+p", "ctrl+k":
		m.fileList.MoveUp()
		return m, nil
	case "down", "ctrl+n", "ctrl+j":
		m.fileList.MoveDown()
		return m, nil
	case "ctrl+u":
		m.fileList.SetFilter("")
		return m, nil
	case "ctrl+w":
		// Delete last word
		input := m.fileList.filter
		input = strings.TrimRight(input, " ")
		if i := strings.LastIndex(input, " "); i >= 0 {
			m.fileList.SetFilter(input[:i+1])
		} else {
			m.fileList.SetFilter("")
		}
		return m, nil
	default:
		ch := msg.String()
		// Ignore the `/` that triggered filter mode
		if len(ch) == 1 && ch != "/" {
			m.fileList.SetFilter(m.fileList.filter + ch)
		}
		return m, nil
	}
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
		target := m.navigator.LinkSelect()
		if target != "" {
			return m, func() tea.Msg {
				return LinkFollowMsg{Target: target}
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
	target := m.navigator.ShowLinks(m.preview.filePath)
	if target != "" {
		// Single link, follow directly
		return m, func() tea.Msg {
			return LinkFollowMsg{Target: target}
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

func (m *Model) handleLinkFollow(target string) (tea.Model, tea.Cmd) {
	// Save current position in history, then push the target so that
	// Back returns the source and Forward from the source reaches the target.
	if m.preview.filePath != "" {
		m.navigator.Navigate(m.preview.filePath, m.preview.scroll)
	}
	m.navigator.Navigate(target, 0)
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

func (m *Model) loadPreview(entry index.FileEntry) (tea.Model, tea.Cmd) {
	// Capture renderers for closure (safe since they're pointers)
	mdRenderer := m.mdRenderer
	codeRenderer := m.codeRenderer

	return m, func() tea.Msg {
		if entry.IsDir {
			return PreviewLoadedMsg{
				Path:    entry.RelPath,
				Content: "[directory]",
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

		if entry.IsMarkdown {
			if mdRenderer != nil {
				rendered, renderErr := mdRenderer.Render(content)
				if renderErr == nil {
					return previewWithSourceMsg{
						preview: PreviewLoadedMsg{
							Path:    entry.RelPath,
							Content: rendered,
						},
						rawSource: content,
					}
				}
			}
		} else {
			ext := filepath.Ext(entry.RelPath)
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
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

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
			plain := strings.ToLower(ansiRe.ReplaceAllString(line, ""))
			if strings.Contains(plain, searchText) {
				m.previewLinks = append(m.previewLinks, previewLink{
					renderedLine: i,
					target:       link.Target,
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

	// Width budget: total must equal m.width exactly.
	// Each BorderRight border costs 1 column.
	// Layout: [listWidth]|[previewWidth]  or  [listWidth]|[previewWidth]|[panelWidth]
	//         border=1                        borders=2
	borders := 1
	panelWidth := 0
	contentHeight := m.height - 1

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

	accentColor := lipgloss.Color("62")  // bright blue-purple for focused
	dimColor := lipgloss.Color("240")    // gray for unfocused
	borderColor := lipgloss.Color("237") // subtle separator

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

	// Vertical separator: one column of │ characters for the full height
	sepStyle := lipgloss.NewStyle().Foreground(borderColor)
	sep := sepStyle.Render(strings.Repeat("│\n", contentHeight-1) + "│")

	// Build each pane as label + body, hard-clipped to exact dimensions.
	// MaxWidth+MaxHeight ensure content never overflows the budget.
	buildPane := func(label, content string, width, height int) string {
		body := lipgloss.NewStyle().
			Width(width).
			MaxWidth(width).
			Height(height).
			MaxHeight(height).
			Render(content)
		return lipgloss.JoinVertical(lipgloss.Left, label, body)
	}

	// File list pane
	listFocused := m.focus == PanelFileList || m.fileList.filtering
	listLabel := paneLabel("FILES", listFocused, listWidth)
	listContent := m.fileList.View()
	left := buildPane(listLabel, listContent, listWidth, bodyHeight)

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

	var main string
	if m.sidePanel.Visible() {
		right := buildPane(previewLabel, previewContent, previewWidth, bodyHeight)

		// Side panel
		sideFocused := m.focus == PanelSide
		sideName := m.sidePanel.TypeName()
		sideLabel := paneLabel(sideName, sideFocused, panelWidth)
		sideContent := m.sidePanel.View()
		side := buildPane(sideLabel, sideContent, panelWidth, bodyHeight)

		main = lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right, sep, side)
	} else {
		right := buildPane(previewLabel, previewContent, previewWidth, bodyHeight)

		main = lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)
	}

	cmdView := m.cmdPalette.View()
	if cmdView != "" {
		return lipgloss.JoinVertical(lipgloss.Left, main, cmdView)
	}

	m.status.width = m.width
	m.status.focus = m.focus
	m.status.showingHelp = m.showingHelp
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
	}
	return textExts[ext]
}

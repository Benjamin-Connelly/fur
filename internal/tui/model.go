package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/lookit/internal/config"
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

	navigator *LinkNavigator
	sidePanel SidePanelModel
	cmdPalette CommandPalette

	focus    Panel
	width    int
	height   int
	quitting bool

	// Track raw markdown source for TOC extraction
	currentRawSource string
}

// New creates a new root TUI model.
func New(cfg *config.Config, idx *index.Index, links *index.LinkGraph) Model {
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

	return Model{
		cfg:          cfg,
		idx:          idx,
		links:        links,
		fileList:     NewFileListModel(idx),
		preview:      NewPreviewModel(),
		status:       NewStatusBarModel(),
		keys:         km,
		mdRenderer:   mdRenderer,
		codeRenderer: codeRenderer,
		navigator:    nav,
		sidePanel:    panel,
		cmdPalette:   palette,
		focus:        PanelFileList,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.fileList.filtering {
			return m.handleFilterKey(msg)
		}
		return m.handleNormalKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		return m, nil

	case FileSelectedMsg:
		return m.loadPreview(msg.Entry)

	case PreviewLoadedMsg:
		m.preview.SetContent(msg.Path, msg.Content)
		m.status.SetFile(msg.Path)
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

func (m Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "tab":
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

	case "/", "ctrl+k":
		m.fileList.StartFilter()
		m.status.SetMode("FILTER")
		return m, nil

	case "?":
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
		m.sidePanel.Toggle(PanelTOC)
		if m.sidePanel.Type() == PanelTOC && m.currentRawSource != "" {
			m.sidePanel.SetTOCFromMarkdown(m.currentRawSource)
		}
		m.recalcLayout()
		return m, nil

	case "b":
		m.sidePanel.Toggle(PanelBacklinks)
		if m.sidePanel.Type() == PanelBacklinks {
			backlinks := m.navigator.BacklinksAt(m.preview.filePath)
			m.sidePanel.SetBacklinks(backlinks)
		}
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
		m.sidePanel.Toggle(PanelBookmarks)
		m.recalcLayout()
		return m, nil

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

func (m Model) handleFileListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		m.fileList.MoveUp()
		return m, nil
	case "down", "j":
		m.fileList.MoveDown()
		return m, nil
	case "enter", "l":
		sel := m.fileList.Selected()
		if sel == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			return FileSelectedMsg{Entry: *sel}
		}
	case "g":
		m.fileList.cursor = 0
		m.fileList.offset = 0
		return m, nil
	case "G":
		if len(m.fileList.filtered) > 0 {
			m.fileList.cursor = len(m.fileList.filtered) - 1
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
	case "home", "g":
		m.preview.scroll = 0
		return m, nil
	case "end", "G":
		m.preview.ScrollToBottom()
		return m, nil
	}
	return m, nil
}

func (m Model) handleSidePanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.fileList.ClearFilter()
		m.status.SetMode("NORMAL")
		return m, nil
	case "enter":
		m.fileList.filtering = false
		m.status.SetMode("NORMAL")
		sel := m.fileList.Selected()
		if sel == nil {
			return m, nil
		}
		return m, func() tea.Msg {
			return FileSelectedMsg{Entry: *sel}
		}
	case "backspace":
		if len(m.fileList.filter) > 0 {
			m.fileList.SetFilter(m.fileList.filter[:len(m.fileList.filter)-1])
		}
		return m, nil
	case "up", "ctrl+p":
		m.fileList.MoveUp()
		return m, nil
	case "down", "ctrl+n":
		m.fileList.MoveDown()
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.fileList.SetFilter(m.fileList.filter + msg.String())
		}
		return m, nil
	}
}

func (m Model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.cmdPalette.Close()
		m.status.SetMode(m.modeString())
		return m, nil
	case "enter":
		// Check for "open" prefix command
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
	case "up", "ctrl+p":
		m.cmdPalette.MoveUp()
		return m, nil
	case "down", "ctrl+n":
		m.cmdPalette.MoveDown()
		return m, nil
	case "backspace":
		if len(m.cmdPalette.input) > 0 {
			m.cmdPalette.SetInput(m.cmdPalette.input[:len(m.cmdPalette.input)-1])
		}
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.cmdPalette.SetInput(m.cmdPalette.input + msg.String())
		}
		return m, nil
	}
}

func (m Model) handleLinkSelectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m Model) handleFollowLink() (tea.Model, tea.Cmd) {
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

func (m Model) handleLinkFollow(target string) (tea.Model, tea.Cmd) {
	// Save current position in history
	if m.preview.filePath != "" {
		m.navigator.Navigate(m.preview.filePath, m.preview.scroll)
	}
	return m.navigateToPath(target, 0)
}

func (m Model) navigateToPath(path string, scroll int) (tea.Model, tea.Cmd) {
	entry := m.idx.Lookup(path)
	if entry == nil {
		return m, func() tea.Msg {
			return StatusMsg{Text: "File not found: " + path}
		}
	}

	// Update file list cursor to match
	for i, e := range m.fileList.filtered {
		if e.RelPath == path {
			m.fileList.cursor = i
			break
		}
	}

	return m, func() tea.Msg {
		return FileSelectedMsg{Entry: *entry}
	}
}

func (m Model) handleCommandLinks() (tea.Model, tea.Cmd) {
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

func (m Model) loadPreview(entry index.FileEntry) (tea.Model, tea.Cmd) {
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
			// Store raw source for TOC extraction (via closure capture of m)
			// We return PreviewLoadedMsg which sets the rendered content;
			// raw source is stored separately via rawSourceMsg
			if m.mdRenderer != nil {
				rendered, renderErr := m.mdRenderer.Render(content)
				if renderErr == nil {
					// Return both raw and rendered via a batch
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
				highlighted, hlErr := m.codeRenderer.Highlight(filepath.Base(entry.RelPath), content)
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

func (m Model) recalcLayout() {
	listWidth := m.width / 3
	previewWidth := m.width - listWidth - 1
	if m.sidePanel.Visible() {
		// Split preview area: 2/3 preview, 1/3 side panel
		panelWidth := previewWidth / 3
		previewWidth = previewWidth - panelWidth - 1
	}
	m.preview.width = previewWidth
	m.preview.height = m.height - 1
	m.fileList.height = m.height - 1
	if m.mdRenderer != nil {
		_ = m.mdRenderer.SetWidth(previewWidth - 2)
	}
}

func (m Model) modeString() string {
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
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Loading..."
	}

	listWidth := m.width / 3
	previewWidth := m.width - listWidth - 1
	panelWidth := 0
	contentHeight := m.height - 1

	if m.sidePanel.Visible() {
		panelWidth = previewWidth / 3
		previewWidth = previewWidth - panelWidth - 1
	}

	listStyle := lipgloss.NewStyle().
		Width(listWidth).
		Height(contentHeight)

	previewStyle := lipgloss.NewStyle().
		Width(previewWidth).
		Height(contentHeight)

	// Focus indicator via border color
	focusColor := lipgloss.Color("63")
	dimColor := lipgloss.Color("240")

	listBorderColor := dimColor
	if m.focus == PanelFileList {
		listBorderColor = focusColor
	}
	listBorder := listStyle.BorderRight(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(listBorderColor)

	left := listBorder.Render(m.fileList.View())

	previewContent := m.preview.View()

	// Overlay link selection on preview if active
	if m.navigator.IsShowingLinks() {
		overlay := m.navigator.LinkOverlayView()
		previewContent = overlay + "\n" + strings.Repeat("─", 20) + "\n" + previewContent
	}

	right := previewStyle.Render(previewContent)

	var main string
	if m.sidePanel.Visible() {
		panelStyle := lipgloss.NewStyle().
			Width(panelWidth).
			Height(contentHeight)

		previewBorderColor := dimColor
		if m.focus == PanelPreview {
			previewBorderColor = focusColor
		}
		rightWithBorder := lipgloss.NewStyle().
			Width(previewWidth).
			Height(contentHeight).
			BorderRight(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(previewBorderColor).
			Render(previewContent)

		panelBorderColor := dimColor
		if m.focus == PanelSide {
			panelBorderColor = focusColor
		}
		_ = panelBorderColor // used for panel styling
		sideContent := panelStyle.Render(m.sidePanel.View())

		main = lipgloss.JoinHorizontal(lipgloss.Top, left, rightWithBorder, sideContent)
	} else {
		_ = right // already rendered
		main = lipgloss.JoinHorizontal(lipgloss.Top, left, previewStyle.Render(previewContent))
	}

	// Command palette overlay at bottom
	cmdView := m.cmdPalette.View()
	if cmdView != "" {
		// Replace status bar with command palette
		return lipgloss.JoinVertical(lipgloss.Left, main, cmdView)
	}

	statusWidth := m.width
	m.status.width = statusWidth
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

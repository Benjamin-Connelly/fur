package tui

import (
	"fmt"
	"strings"

	"github.com/Benjamin-Connelly/fur/internal/git"
	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/Benjamin-Connelly/fur/internal/render"
)

// PanelType identifies the type of side panel displayed.
type PanelType int

const (
	PanelNone PanelType = iota
	PanelTOC
	PanelBacklinks
	PanelGitInfo
	PanelBookmarks
)

// TOCEntry represents a heading in the table of contents.
type TOCEntry struct {
	Level int
	Text  string
	Line  int
}

// Bookmark represents a saved location.
type Bookmark struct {
	Path   string
	Title  string
	Scroll int
}

// PanelSelectMsg is sent when the user selects an item in a side panel.
type PanelSelectMsg struct {
	Path   string // for backlinks/bookmarks: file to navigate to
	Line   int    // for TOC: line to scroll to
	Scroll int    // for bookmarks: scroll position to restore
}

// SidePanelModel manages the optional side panel (TOC, backlinks, etc.).
type SidePanelModel struct {
	panelType PanelType
	toc       []TOCEntry
	backlinks []index.Link
	bookmarks []Bookmark
	gitInfo   string
	cursor    int
}

// NewSidePanelModel creates a side panel model.
func NewSidePanelModel() SidePanelModel {
	return SidePanelModel{
		panelType: PanelNone,
	}
}

// Toggle switches between showing and hiding the given panel type.
func (m *SidePanelModel) Toggle(pt PanelType) {
	if m.panelType == pt {
		m.panelType = PanelNone
	} else {
		m.panelType = pt
		m.cursor = 0
	}
}

// SetTOC updates the table of contents entries.
func (m *SidePanelModel) SetTOC(entries []TOCEntry) {
	m.toc = entries
}

// SetTOCFromMarkdown extracts headings from raw markdown source and updates TOC.
func (m *SidePanelModel) SetTOCFromMarkdown(source string) {
	headings := render.ExtractHeadings(source)
	entries := make([]TOCEntry, len(headings))
	for i, h := range headings {
		entries[i] = TOCEntry{Level: h.Level, Text: h.Text, Line: h.Line}
	}
	m.toc = entries
}

// SetBacklinks updates the backlinks list from the link graph.
func (m *SidePanelModel) SetBacklinks(links []index.Link) {
	m.backlinks = links
}

// SetGitInfo updates the git info panel content.
func (m *SidePanelModel) SetGitInfo(repoRoot, filePath string) {
	var b strings.Builder

	repo, err := git.Open(repoRoot)
	if err != nil {
		m.gitInfo = "Not a git repository"
		return
	}

	branch, err := repo.Branch()
	if err == nil {
		fmt.Fprintf(&b, "Branch: %s\n", branch)
	}

	clean, err := repo.IsClean()
	if err == nil {
		if clean {
			b.WriteString("Status: clean\n")
		} else {
			b.WriteString("Status: dirty\n")
		}
	}

	if filePath != "" {
		fs, err := repo.FileStatusAt(filePath)
		if err == nil {
			staging := string([]byte{byte(fs.Staging)})
			worktree := string([]byte{byte(fs.Worktree)})
			if staging == " " && worktree == " " {
				b.WriteString("File: unmodified\n")
			} else {
				fmt.Fprintf(&b, "File: %s%s\n", staging, worktree)
			}
		}
	}

	commits, err := repo.Log(1)
	if err == nil && len(commits) > 0 {
		c := commits[0]
		b.WriteString("\nLast commit:\n")
		fmt.Fprintf(&b, "  %s\n", c.Hash[:8])
		fmt.Fprintf(&b, "  %s\n", c.Author)
		fmt.Fprintf(&b, "  %s\n", c.Date.Format("2006-01-02 15:04"))
		msg := c.Message
		if i := strings.Index(msg, "\n"); i >= 0 {
			msg = msg[:i]
		}
		fmt.Fprintf(&b, "  %s\n", msg)
	}

	m.gitInfo = b.String()
}

// AddBookmark adds a bookmark.
func (m *SidePanelModel) AddBookmark(bm Bookmark) {
	// Avoid duplicates for same path
	for i, existing := range m.bookmarks {
		if existing.Path == bm.Path {
			m.bookmarks[i] = bm
			return
		}
	}
	m.bookmarks = append(m.bookmarks, bm)
}

// RemoveBookmark removes a bookmark by index.
func (m *SidePanelModel) RemoveBookmark(idx int) {
	if idx >= 0 && idx < len(m.bookmarks) {
		m.bookmarks = append(m.bookmarks[:idx], m.bookmarks[idx+1:]...)
	}
}

// Visible returns whether the side panel is currently shown.
func (m *SidePanelModel) Visible() bool {
	return m.panelType != PanelNone
}

// Type returns the current panel type.
func (m *SidePanelModel) Type() PanelType {
	return m.panelType
}

// TypeName returns a display name for the current panel type.
func (m *SidePanelModel) TypeName() string {
	switch m.panelType {
	case PanelTOC:
		return "TOC"
	case PanelBacklinks:
		return "BACKLINKS"
	case PanelBookmarks:
		return "BOOKMARKS"
	case PanelGitInfo:
		return "GIT"
	default:
		return "PANEL"
	}
}

// MoveUp moves the panel cursor up.
func (m *SidePanelModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

// MoveDown moves the panel cursor down.
func (m *SidePanelModel) MoveDown() {
	max := m.itemCount() - 1
	if max < 0 {
		max = 0
	}
	if m.cursor < max {
		m.cursor++
	}
}

func (m *SidePanelModel) itemCount() int {
	switch m.panelType {
	case PanelTOC:
		return len(m.toc)
	case PanelBacklinks:
		return len(m.backlinks)
	case PanelBookmarks:
		return len(m.bookmarks)
	default:
		return 0
	}
}

// Select returns information about the currently selected item.
func (m *SidePanelModel) Select() *PanelSelectMsg {
	switch m.panelType {
	case PanelTOC:
		if m.cursor >= 0 && m.cursor < len(m.toc) {
			return &PanelSelectMsg{Line: m.toc[m.cursor].Line}
		}
	case PanelBacklinks:
		if m.cursor >= 0 && m.cursor < len(m.backlinks) {
			return &PanelSelectMsg{Path: m.backlinks[m.cursor].Source}
		}
	case PanelBookmarks:
		if m.cursor >= 0 && m.cursor < len(m.bookmarks) {
			bm := m.bookmarks[m.cursor]
			return &PanelSelectMsg{Path: bm.Path, Scroll: bm.Scroll}
		}
	}
	return nil
}

// View renders the side panel.
func (m SidePanelModel) View() string {
	switch m.panelType {
	case PanelTOC:
		return m.viewTOC()
	case PanelBacklinks:
		return m.viewBacklinks()
	case PanelBookmarks:
		return m.viewBookmarks()
	case PanelGitInfo:
		return m.viewGitInfo()
	default:
		return ""
	}
}

func (m SidePanelModel) viewTOC() string {
	if len(m.toc) == 0 {
		return "No headings found"
	}
	var s string
	s += "Table of Contents\n"
	s += strings.Repeat("─", 20) + "\n"
	for i, entry := range m.toc {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		indent := ""
		for j := 1; j < entry.Level; j++ {
			indent += "  "
		}
		s += cursor + indent + entry.Text + "\n"
	}
	return s
}

func (m SidePanelModel) viewBacklinks() string {
	if len(m.backlinks) == 0 {
		return "No backlinks"
	}
	var s string
	s += "Backlinks\n"
	s += strings.Repeat("─", 20) + "\n"
	for i, link := range m.backlinks {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		label := link.Source
		if link.Text != "" {
			label = fmt.Sprintf("%s (%s)", link.Source, link.Text)
		}
		s += cursor + label + "\n"
	}
	return s
}

func (m SidePanelModel) viewBookmarks() string {
	if len(m.bookmarks) == 0 {
		return "No bookmarks"
	}
	var s string
	s += "Bookmarks\n"
	s += strings.Repeat("─", 20) + "\n"
	for i, bm := range m.bookmarks {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		s += cursor + bm.Title + " (" + bm.Path + ")\n"
	}
	return s
}

func (m SidePanelModel) viewGitInfo() string {
	if m.gitInfo == "" {
		return "No git info available"
	}
	s := "Git Info\n"
	s += strings.Repeat("─", 20) + "\n"
	s += m.gitInfo
	return s
}

package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/fur/internal/index"
)

// treeLess compares two file entries for proper tree ordering:
// directories before files at each level, alphabetical within type,
// parent directories always before their children.
func treeLess(a, b index.FileEntry) bool {
	ap := strings.Split(a.RelPath, string(filepath.Separator))
	bp := strings.Split(b.RelPath, string(filepath.Separator))

	// Compare segment by segment
	minLen := len(ap)
	if len(bp) < minLen {
		minLen = len(bp)
	}

	for i := 0; i < minLen; i++ {
		if ap[i] == bp[i] {
			continue
		}
		// At this level, both entries diverge. If one is the last segment
		// (leaf) and the other isn't (has children = is a dir at this level),
		// the directory comes first.
		aIsLeaf := i == len(ap)-1
		bIsLeaf := i == len(bp)-1

		aIsDir := a.IsDir || !aIsLeaf
		bIsDir := b.IsDir || !bIsLeaf

		if aIsDir != bIsDir {
			return aIsDir
		}
		return strings.ToLower(ap[i]) < strings.ToLower(bp[i])
	}

	// One path is a prefix of the other; the shorter (parent) comes first
	return len(ap) < len(bp)
}

// treeNode represents a file or directory in the tree.
type treeNode struct {
	entry    index.FileEntry
	name     string // display name (basename)
	depth    int
	isDir    bool
	expanded bool
}

// FileListModel manages the file list panel with fuzzy filtering.
type FileListModel struct {
	idx        index.Indexer
	entries    []index.FileEntry
	tree       []treeNode        // full tree
	visible    []treeNode        // visible nodes (respecting collapsed dirs)
	filtered   []index.FileEntry // for fuzzy search results
	collapsed  map[string]bool   // collapsed directory paths
	cursor     int
	filter     string
	filtering  bool
	searchMode string // "filename" or "content" — set by parent Model
	offset     int    // scroll offset
	height     int
}

// NewFileListModel creates a file list panel.
func NewFileListModel(idx index.Indexer) FileListModel {
	entries := idx.Entries()
	collapsed := make(map[string]bool)
	// Start with all directories collapsed
	for _, e := range entries {
		if e.IsDir {
			collapsed[e.RelPath] = true
		}
	}
	m := FileListModel{
		idx:       idx,
		entries:   entries,
		filtered:  entries,
		collapsed: collapsed,
	}
	m.buildTree()
	return m
}

func (m *FileListModel) buildTree() {
	// Sort entries into proper tree order: within each directory,
	// directories before files, then alphabetical. Parent dirs
	// always appear before their children.
	sorted := make([]index.FileEntry, len(m.entries))
	copy(sorted, m.entries)
	sort.Slice(sorted, func(i, j int) bool {
		return treeLess(sorted[i], sorted[j])
	})

	m.tree = nil
	for _, e := range sorted {
		depth := strings.Count(e.RelPath, string(filepath.Separator))
		name := filepath.Base(e.RelPath)
		m.tree = append(m.tree, treeNode{
			entry:    e,
			name:     name,
			depth:    depth,
			isDir:    e.IsDir,
			expanded: !m.collapsed[e.RelPath],
		})
	}
	m.rebuildVisible()
}

func (m *FileListModel) rebuildVisible() {
	m.visible = nil
	for _, node := range m.tree {
		// Check if any ancestor is collapsed
		if m.isAncestorCollapsed(node.entry.RelPath) {
			continue
		}
		node.expanded = !m.collapsed[node.entry.RelPath]
		m.visible = append(m.visible, node)
	}
}

func (m *FileListModel) isAncestorCollapsed(relPath string) bool {
	parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
	path := ""
	for _, part := range parts {
		if part == "." {
			continue
		}
		if path == "" {
			path = part
		} else {
			path = path + string(filepath.Separator) + part
		}
		if m.collapsed[path] {
			return true
		}
	}
	return false
}

// ToggleDir collapses or expands the directory at the cursor.
func (m *FileListModel) ToggleDir() {
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return
	}
	node := m.visible[m.cursor]
	if !node.isDir {
		return
	}
	path := node.entry.RelPath
	if m.collapsed[path] {
		delete(m.collapsed, path)
	} else {
		m.collapsed[path] = true
	}
	m.rebuildVisible()
	// Keep cursor in bounds
	if len(m.visible) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.visible) {
		m.cursor = len(m.visible) - 1
	}
}

// SelectedVisible returns the entry for the current cursor in tree mode.
func (m *FileListModel) SelectedVisible() *index.FileEntry {
	if m.filtering {
		return m.Selected()
	}
	if len(m.visible) == 0 || m.cursor >= len(m.visible) {
		return nil
	}
	return &m.visible[m.cursor].entry
}

// Selected returns the currently selected entry from filtered list.
func (m *FileListModel) Selected() *index.FileEntry {
	if len(m.filtered) == 0 {
		return nil
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	return &m.filtered[m.cursor]
}

// SetFilter updates the fuzzy filter query and refilters.
func (m *FileListModel) SetFilter(query string) {
	m.filter = query
	if query == "" {
		m.filtered = m.entries
	} else {
		m.filtered = m.idx.FuzzySearch(query, 100)
	}
	m.cursor = 0
	m.offset = 0
}

// StartFilter enters filter mode.
func (m *FileListModel) StartFilter() {
	m.filtering = true
	m.filter = ""
}

// ClearFilter exits filter mode and resets the list.
func (m *FileListModel) ClearFilter() {
	m.filtering = false
	m.filter = ""
	m.filtered = m.entries
	m.cursor = 0
	m.offset = 0
}

// SelectByPath positions the cursor on the file with the given relative path.
// It expands any collapsed parent directories so the file is visible.
func (m *FileListModel) SelectByPath(relPath string) {
	// Expand parent directories so the file becomes visible
	parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
	path := ""
	for _, part := range parts {
		if part == "." {
			continue
		}
		if path == "" {
			path = part
		} else {
			path = path + string(filepath.Separator) + part
		}
		delete(m.collapsed, path)
	}
	m.rebuildVisible()

	// Find the file in the visible tree
	for i, node := range m.visible {
		if node.entry.RelPath == relPath {
			m.cursor = i
			return
		}
	}
}

// MoveUp moves the cursor up.
func (m *FileListModel) MoveUp() {
	if m.cursor > 0 {
		m.cursor--
		if m.cursor < m.offset {
			m.offset = m.cursor
		}
	}
}

// MoveDown moves the cursor down.
func (m *FileListModel) MoveDown() {
	max := m.listLen() - 1
	if m.cursor < max {
		m.cursor++
		visible := m.visibleRows()
		if visible > 0 && m.cursor >= m.offset+visible {
			m.offset = m.cursor - visible + 1
		}
	}
}

func (m *FileListModel) listLen() int {
	if m.filtering || m.filter != "" {
		return len(m.filtered)
	}
	return len(m.visible)
}

func (m *FileListModel) visibleRows() int {
	rows := m.height
	if m.filtering {
		rows -= 2 // filter input + separator
	}
	if rows < 1 {
		rows = 20
	}
	return rows
}

// View renders the file list.
func (m FileListModel) View() string {
	if m.filtering || m.filter != "" {
		return m.viewFiltered()
	}
	return m.viewTree()
}

func (m FileListModel) viewTree() string {
	var b strings.Builder

	if len(m.visible) == 0 {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		return dim.Render("No files found")
	}

	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.visible) {
		end = len(m.visible)
	}

	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Bold(true)
	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).
		Bold(true)
	mdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("114"))
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	for i := m.offset; i < end; i++ {
		node := m.visible[i]
		indent := strings.Repeat("  ", node.depth)

		var icon string
		if node.isDir {
			if m.collapsed[node.entry.RelPath] {
				icon = "▸ 📁"
			} else {
				icon = "▾ 📁"
			}
		} else if node.entry.IsMarkdown {
			icon = "  📝"
		} else {
			icon = "   "
		}

		line := fmt.Sprintf("%s%s %s", indent, icon, node.name)

		if i == m.cursor {
			b.WriteString(cursorStyle.Render(line) + "\n")
		} else if node.isDir {
			b.WriteString(dirStyle.Render(line) + "\n")
		} else if node.entry.IsMarkdown {
			b.WriteString(mdStyle.Render(line) + "\n")
		} else {
			b.WriteString(normalStyle.Render(line) + "\n")
		}
	}

	// Scroll indicator
	if len(m.visible) > visible {
		pct := 0
		if len(m.visible)-visible > 0 {
			pct = m.offset * 100 / (len(m.visible) - visible)
		}
		indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(indicator.Render(fmt.Sprintf(" %d/%d (%d%%)", m.cursor+1, len(m.visible), pct)))
	}

	return b.String()
}

func (m FileListModel) viewFiltered() string {
	var b strings.Builder

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))
	modeLabel := m.searchMode
	if modeLabel == "" {
		modeLabel = "filename"
	}
	if m.filtering {
		b.WriteString(filterStyle.Render("/ "+m.filter+"_ ("+modeLabel+")") + "\n")
	} else {
		frozenStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(frozenStyle.Render("/ "+m.filter+" ("+modeLabel+", esc to clear)") + "\n")
	}
	b.WriteString(strings.Repeat("─", 20) + "\n")

	if len(m.filtered) == 0 {
		dim := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		if m.filter != "" {
			b.WriteString(dim.Render("No matches for: " + m.filter))
		} else {
			b.WriteString(dim.Render("No files found"))
		}
		return b.String()
	}

	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("62")).
		Bold(true)
	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))
	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75"))
	mdStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("114"))

	for i := m.offset; i < end; i++ {
		entry := m.filtered[i]
		icon := "  "
		if entry.IsDir {
			icon = "📁"
		} else if entry.IsMarkdown {
			icon = "📝"
		}

		line := fmt.Sprintf(" %s %s", icon, entry.RelPath)

		if i == m.cursor {
			b.WriteString(cursorStyle.Render("▸"+line) + "\n")
		} else if entry.IsDir {
			b.WriteString(dirStyle.Render(" "+line) + "\n")
		} else if entry.IsMarkdown {
			b.WriteString(mdStyle.Render(" "+line) + "\n")
		} else {
			b.WriteString(normalStyle.Render(" "+line) + "\n")
		}
	}

	if len(m.filtered) > visible {
		pct := 0
		if len(m.filtered)-visible > 0 {
			pct = m.offset * 100 / (len(m.filtered) - visible)
		}
		indicator := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		b.WriteString(indicator.Render(fmt.Sprintf(" %d/%d (%d%%)", m.cursor+1, len(m.filtered), pct)))
	}

	return b.String()
}

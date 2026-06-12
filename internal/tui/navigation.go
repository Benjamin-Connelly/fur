package tui

import (
	"fmt"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/Benjamin-Connelly/fur/internal/render"
	"github.com/Benjamin-Connelly/fur/internal/theme"
)

// ansiRe strips ANSI escape sequences for plain-text search.
// Covers SGR, CSI, OSC, and hyperlink sequences that Glamour/lipgloss emit.
var ansiRe = regexp.MustCompile(`\x1b(?:\[[0-9;]*[a-zA-Z]|\]8;[^;]*;[^\x1b]*\x1b\\|\][^\x07]*\x07)`)

// stripANSI removes all ANSI escape sequences from s.
func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
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
	fmt.Fprintf(&b, "Links in %s\n", m.preview.filePath)
	b.WriteString(strings.Repeat("=", 40) + "\n\n")
	for _, link := range links {
		status := " "
		if link.Broken {
			status = "!"
		}
		fmt.Fprintf(&b, "  [%s] %s -> %s", status, link.Text, link.Target)
		b.WriteString("\n")
	}
	content := b.String()
	return m, func() tea.Msg {
		return PreviewLoadedMsg{Path: "Links: " + m.preview.filePath, Content: content}
	}
}

// scrollToFragment finds a heading matching the fragment slug and scrolls to it.
// Uses GitHub-style duplicate disambiguation: "heading", "heading-1", "heading-2".
func (m *Model) scrollToFragment(fragment, rawSource string) {
	headings := render.ExtractHeadings(rawSource)
	slugs := render.AnchorSlugs(rawSource)
	slug := strings.ToLower(fragment)

	// Match against the centralized slug assignment so the TUI resolves the
	// same heading that the web TOC/API would for an identical fragment
	// (audit Chain M). occurrence is the 0-based index of this heading among
	// those sharing its base slug, used to locate the right rendered line.
	type match struct {
		text       string
		occurrence int
	}
	var candidates []match
	baseCounts := make(map[string]int)

	for i, h := range headings {
		base := slugify(h.Text)
		occ := baseCounts[base]
		baseCounts[base]++
		if slugs[i] == slug {
			candidates = append(candidates, match{text: h.Text, occurrence: occ})
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
		data, err := afero.ReadFile(m.idx.Fs(), f.Path)
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

// slugify converts heading text to a URL-compatible anchor slug.
// Delegates to render.Slugify for consistency across TUI and index.
func slugify(s string) string {
	return render.Slugify(s)
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

// cycleTheme advances to the next theme in theme.CycleOrder and re-renders the
// open preview at the current width, preserving scroll position.
func (m *Model) cycleTheme() (*Model, tea.Cmd) {
	return m, m.SetTheme(theme.Next(m.cfg.Theme))
}

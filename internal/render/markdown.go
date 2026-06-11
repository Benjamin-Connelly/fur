package render

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/afero"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/Benjamin-Connelly/fur/internal/theme"
)

// Heading represents a markdown heading extracted from source.
type Heading struct {
	Level int
	Text  string
	Line  int
}

// Link represents a markdown link extracted from source.
type Link struct {
	Text        string
	Destination string
	Line        int
}

// MarkdownRenderer wraps Glamour for TUI markdown rendering.
type MarkdownRenderer struct {
	renderer *glamour.TermRenderer
	theme    string
	width    int
	wikiFg   string // wikilink foreground (palette teal), empty for ascii
	fs       afero.Fs
}

// NewMarkdownRenderer creates a markdown renderer with the given theme and width.
func NewMarkdownRenderer(themeName string, width int) (*MarkdownRenderer, error) {
	opts := []glamour.TermRendererOption{glamour.WithWordWrap(width)}
	wikiFg := ""
	if themeName == "ascii" {
		opts = append(opts, glamour.WithStandardStyle("notty"))
	} else {
		p := theme.Resolve(themeName)
		opts = append(opts, glamour.WithStyles(theme.GlamourStyle(p)))
		wikiFg = p.Teal
	}

	r, err := glamour.NewTermRenderer(opts...)
	if err != nil {
		return nil, err
	}

	return &MarkdownRenderer{
		renderer: r,
		theme:    themeName,
		width:    width,
		wikiFg:   wikiFg,
		fs:       afero.NewOsFs(),
	}, nil
}

// SetFs sets the filesystem for file operations.
func (r *MarkdownRenderer) SetFs(fs afero.Fs) {
	r.fs = fs
}

// wikiLinkRe matches [[target]] and [[target|display]] in rendered output.
var wikiLinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// highlightWikilinks colorizes wikilink syntax in rendered output.
func highlightWikilinks(rendered, fg string) string {
	linkStyle := lipgloss.NewStyle().Bold(true)
	if fg != "" {
		linkStyle = linkStyle.Foreground(lipgloss.Color(fg))
	}

	return wikiLinkRe.ReplaceAllStringFunc(rendered, func(match string) string {
		inner := match[2 : len(match)-2] // strip [[ and ]]
		// Show display text for [[target|display]] syntax
		display := inner
		if i := strings.Index(inner, "|"); i >= 0 {
			display = inner[i+1:]
		}
		return linkStyle.Render("⟦" + display + "⟧")
	})
}

// Render converts markdown to styled terminal output.
// On error, returns the raw source as fallback.
func (r *MarkdownRenderer) Render(source string) (string, error) {
	// Glamour preserves source soft-breaks inside list items; unwrap them so
	// list text reflows to the pane width like ordinary paragraphs.
	source = unwrapSoftBreaks(source)
	out, err := r.renderer.Render(source)
	if err != nil {
		return source, nil
	}
	out = spaceListItems(out)
	out = highlightWikilinks(out, r.wikiFg)
	return out, nil
}

// RenderFile reads a file and renders its markdown content.
// On render error, returns the raw file content as fallback.
func (r *MarkdownRenderer) RenderFile(filePath string) (string, error) {
	data, err := afero.ReadFile(r.fs, filePath)
	if err != nil {
		return "", err
	}
	return r.Render(string(data))
}

// SetWidth updates the word wrap width and recreates the renderer.
func (r *MarkdownRenderer) SetWidth(width int) error {
	r.width = width
	nr, err := NewMarkdownRenderer(r.theme, width)
	if err != nil {
		return err
	}
	r.renderer = nr.renderer
	return nil
}

// parseMarkdown parses source into a goldmark AST.
func parseMarkdown(source []byte) ast.Node {
	md := goldmark.New()
	reader := text.NewReader(source)
	return md.Parser().Parse(reader)
}

// lineNumber returns the 1-based line number for a byte offset in source.
func lineNumber(source []byte, offset int) int {
	if offset > len(source) {
		offset = len(source)
	}
	return bytes.Count(source[:offset], []byte("\n")) + 1
}

// nodeText extracts the text content of a node from source.
func nodeText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			buf.Write(t.Segment.Value(source))
		}
	}
	return buf.String()
}

// nodeStartOffset returns the byte offset where a node starts in source.
func nodeStartOffset(n ast.Node) int {
	if n.Type() == ast.TypeBlock {
		if bl, ok := n.(interface{ Lines() *text.Segments }); ok {
			if bl.Lines().Len() > 0 {
				return bl.Lines().At(0).Start
			}
		}
	}
	// For inline nodes, walk children to find first text segment
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			return t.Segment.Start
		}
	}
	return 0
}

// ExtractHeadings returns all headings from markdown source.
func ExtractHeadings(source string) []Heading {
	src := []byte(source)
	doc := parseMarkdown(src)

	var headings []Heading
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if h, ok := n.(*ast.Heading); ok {
			headings = append(headings, Heading{
				Level: h.Level,
				Text:  nodeText(h, src),
				Line:  lineNumber(src, nodeStartOffset(h)),
			})
		}
		return ast.WalkContinue, nil
	})
	return headings
}

// Slugify converts heading text to a URL-compatible anchor slug.
// Matches GitHub's heading anchor generation: lowercase, spaces to hyphens,
// strip non-alphanumeric except hyphens and underscores.
func Slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// HeadingSlugs returns a map of slug -> true for all headings in the source.
// Duplicate headings get GitHub-style suffixes: "heading", "heading-1", "heading-2".
func HeadingSlugs(source string) map[string]bool {
	headings := ExtractHeadings(source)
	slugs := make(map[string]bool, len(headings))
	counts := make(map[string]int)
	for _, h := range headings {
		base := Slugify(h.Text)
		n := counts[base]
		counts[base]++
		slug := base
		if n > 0 {
			slug = base + "-" + strconv.Itoa(n)
		}
		slugs[slug] = true
	}
	return slugs
}

// ExtractLinks returns all links from markdown source.
func ExtractLinks(source string) []Link {
	src := []byte(source)
	doc := parseMarkdown(src)

	var links []Link
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if l, ok := n.(*ast.Link); ok {
			links = append(links, Link{
				Text:        nodeText(l, src),
				Destination: string(l.Destination),
				Line:        lineNumber(src, nodeStartOffset(l)),
			})
		}
		return ast.WalkContinue, nil
	})
	return links
}

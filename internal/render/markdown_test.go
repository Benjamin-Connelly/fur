package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Getting Started", "getting-started"},
		{"API v2.0 Release!", "api-v20-release"},
		{"multiple   spaces", "multiple---spaces"},
		{"under_score", "under_score"},
		{"ALLCAPS", "allcaps"},
		{"", ""},
		{"123 Numbers", "123-numbers"},
		{"special!@#$%chars", "specialchars"},
		{"hyphen-already", "hyphen-already"},
		{"Unicode café résumé", "unicode-caf-rsum"},
		{"  leading trailing  ", "--leading-trailing--"},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHeadingSlugs(t *testing.T) {
	source := `# Introduction
## Getting Started
## Getting Started
### Details
## Getting Started
`
	slugs := HeadingSlugs(source)

	// First occurrence: "getting-started"
	if !slugs["getting-started"] {
		t.Error("expected slug 'getting-started'")
	}
	// Second occurrence: "getting-started-1"
	if !slugs["getting-started-1"] {
		t.Error("expected slug 'getting-started-1'")
	}
	// Third occurrence: "getting-started-2"
	if !slugs["getting-started-2"] {
		t.Error("expected slug 'getting-started-2'")
	}
	if !slugs["introduction"] {
		t.Error("expected slug 'introduction'")
	}
	if !slugs["details"] {
		t.Error("expected slug 'details'")
	}
	// Sanity: non-existent slug
	if slugs["nonexistent"] {
		t.Error("unexpected slug 'nonexistent'")
	}
}

func TestHeadingSlugs_Empty(t *testing.T) {
	slugs := HeadingSlugs("no headings here")
	if len(slugs) != 0 {
		t.Errorf("expected 0 slugs, got %d", len(slugs))
	}
}

func TestNewMarkdownRenderer(t *testing.T) {
	themes := []string{"dark", "light", "auto", "ascii"}
	for _, theme := range themes {
		t.Run(theme, func(t *testing.T) {
			r, err := NewMarkdownRenderer(theme, 80)
			if err != nil {
				t.Fatalf("NewMarkdownRenderer(%q, 80) error: %v", theme, err)
			}
			if r == nil {
				t.Fatal("expected non-nil renderer")
			}
		})
	}
}

func TestRender_Basic(t *testing.T) {
	r, err := NewMarkdownRenderer("dark", 80)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer: %v", err)
	}
	out, err := r.Render("# Hello\nWorld")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(strings.TrimSpace(out)) == 0 {
		t.Error("expected non-empty rendered output")
	}
}

func TestRender_Wikilinks(t *testing.T) {
	r, err := NewMarkdownRenderer("dark", 80)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer: %v", err)
	}
	out, err := r.Render("Check [[target]] for details")
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	// Wikilinks get converted to styled output with ⟦⟧ brackets
	if !strings.Contains(out, "target") {
		t.Error("expected rendered output to contain 'target'")
	}
}

func TestRenderFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	os.WriteFile(path, []byte("# Test File\nSome content."), 0o644)

	r, err := NewMarkdownRenderer("dark", 80)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer: %v", err)
	}
	out, err := r.RenderFile(path)
	if err != nil {
		t.Fatalf("RenderFile: %v", err)
	}
	if len(strings.TrimSpace(out)) == 0 {
		t.Error("expected non-empty output from RenderFile")
	}
}

func TestRenderFile_NotFound(t *testing.T) {
	r, err := NewMarkdownRenderer("dark", 80)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer: %v", err)
	}
	_, err = r.RenderFile("/nonexistent/file.md")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestSetWidth(t *testing.T) {
	r, err := NewMarkdownRenderer("dark", 80)
	if err != nil {
		t.Fatalf("NewMarkdownRenderer: %v", err)
	}
	if err := r.SetWidth(120); err != nil {
		t.Fatalf("SetWidth: %v", err)
	}
	// Verify renderer still works after width change
	out, err := r.Render("# After resize")
	if err != nil {
		t.Fatalf("Render after SetWidth: %v", err)
	}
	if len(strings.TrimSpace(out)) == 0 {
		t.Error("expected non-empty output after SetWidth")
	}
}

func TestExtractHeadings(t *testing.T) {
	source := "# Title\n\nParagraph.\n\n## Section A\n\n### Subsection\n\n## Section B\n"
	headings := ExtractHeadings(source)

	if len(headings) != 4 {
		t.Fatalf("expected 4 headings, got %d", len(headings))
	}

	expected := []struct {
		level int
		text  string
	}{
		{1, "Title"},
		{2, "Section A"},
		{3, "Subsection"},
		{2, "Section B"},
	}

	for i, e := range expected {
		if headings[i].Level != e.level {
			t.Errorf("heading %d: level=%d, want %d", i, headings[i].Level, e.level)
		}
		if headings[i].Text != e.text {
			t.Errorf("heading %d: text=%q, want %q", i, headings[i].Text, e.text)
		}
		if headings[i].Line < 1 {
			t.Errorf("heading %d: line=%d, want >= 1", i, headings[i].Line)
		}
	}
}

func TestExtractLinks(t *testing.T) {
	source := "Check [Google](https://google.com) and [Docs](./docs.md).\n"
	links := ExtractLinks(source)

	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Text != "Google" {
		t.Errorf("link 0 text=%q, want 'Google'", links[0].Text)
	}
	if links[0].Destination != "https://google.com" {
		t.Errorf("link 0 dest=%q, want 'https://google.com'", links[0].Destination)
	}
	if links[1].Text != "Docs" {
		t.Errorf("link 1 text=%q, want 'Docs'", links[1].Text)
	}
	if links[1].Destination != "./docs.md" {
		t.Errorf("link 1 dest=%q, want './docs.md'", links[1].Destination)
	}
	if links[0].Line != 1 {
		t.Errorf("link 0 line=%d, want 1", links[0].Line)
	}
}

func TestResolveTheme(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"light", "light"},
		{"ascii", "notty"},
		{"dark", "dark"},
		{"unknown", "dark"},
		{"", "dark"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := resolveTheme(tt.input)
			if got != tt.want {
				t.Errorf("resolveTheme(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHighlightWikilinks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
		desc  string
	}{
		{
			"simple wikilink",
			"See [[target]] here",
			func(s string) bool { return strings.Contains(s, "target") && !strings.Contains(s, "[[target]]") },
			"should replace [[target]] with styled output",
		},
		{
			"display text",
			"See [[target|Display Text]] here",
			func(s string) bool { return strings.Contains(s, "Display Text") },
			"should show display text from [[target|Display Text]]",
		},
		{
			"no wikilinks",
			"Plain text without links",
			func(s string) bool { return s == "Plain text without links" },
			"should return unchanged text",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightWikilinks(tt.input)
			if !tt.check(got) {
				t.Errorf("%s: got %q", tt.desc, got)
			}
		})
	}
}

func TestLineNumber(t *testing.T) {
	source := []byte("line1\nline2\nline3\n")
	tests := []struct {
		offset int
		want   int
	}{
		{0, 1},
		{3, 1},
		{5, 1},   // newline char itself
		{6, 2},   // start of line2
		{11, 2},  // newline after line2
		{12, 3},  // start of line3
		{100, 4}, // past end, clamped
	}
	for _, tt := range tests {
		got := lineNumber(source, tt.offset)
		if got != tt.want {
			t.Errorf("lineNumber(source, %d) = %d, want %d", tt.offset, got, tt.want)
		}
	}
}

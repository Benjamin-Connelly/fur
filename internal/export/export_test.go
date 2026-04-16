package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/index"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

func TestTitleFromPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"README.md", "README"},
		{"my-doc.md", "my doc"},
		{"docs/guide.md", "guide"},
		{"notes.markdown", "notes"},
	}
	for _, tt := range tests {
		got := titleFromPath(tt.input)
		if got != tt.want {
			t.Errorf("titleFromPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReplaceExt(t *testing.T) {
	tests := []struct {
		name, newExt, want string
	}{
		{"file.md", ".html", "file.html"},
		{"doc.markdown", ".html", "doc.html"},
		{"README.md", ".pdf", "README.pdf"},
	}
	for _, tt := range tests {
		got := replaceExt(tt.name, tt.newExt)
		if got != tt.want {
			t.Errorf("replaceExt(%q, %q) = %q, want %q", tt.name, tt.newExt, got, tt.want)
		}
	}
}

func TestUnescapeHTML(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"&amp;", "&"},
		{"&lt;div&gt;", "<div>"},
		{"&quot;hello&quot;", `"hello"`},
		{"it&#39;s", "it's"},
		{"&#34;quoted&#34;", `"quoted"`},
		{"no entities", "no entities"},
	}
	for _, tt := range tests {
		got := unescapeHTML(tt.input)
		if got != tt.want {
			t.Errorf("unescapeHTML(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- New comprehensive tests ---

func buildIndex(t *testing.T, dir string) *index.Index {
	t.Helper()
	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("index.Build: %v", err)
	}
	return idx
}

func TestExport_HTMLSingleFile(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "test.md"), []byte("# Hello\n\nWorld\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx := buildIndex(t, srcDir)

	err := Export(idx, Options{
		Format:    FormatHTML,
		OutputDir: outDir,
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	outFile := filepath.Join(outDir, "test.html")
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	html := string(data)
	for _, want := range []string{"<!DOCTYPE html>", "<html", "<head>", "<body>", "<article", "Hello", "World"} {
		if !strings.Contains(html, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestExport_HTMLMultipleFiles(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	files := map[string]string{
		"one.md":   "# One\n",
		"two.md":   "# Two\n",
		"three.md": "# Three\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	idx := buildIndex(t, srcDir)
	if err := Export(idx, Options{Format: FormatHTML, OutputDir: outDir}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	for name := range files {
		htmlName := replaceExt(name, ".html")
		if _, err := os.Stat(filepath.Join(outDir, htmlName)); err != nil {
			t.Errorf("missing output for %s: %v", name, err)
		}
	}
}

func TestExportHTML_Structure(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	source := []byte("# Title\n\nSome paragraph.\n")
	if err := os.WriteFile(filepath.Join(srcDir, "doc.md"), source, 0o644); err != nil {
		t.Fatal(err)
	}

	absPath := filepath.Join(srcDir, "doc.md")
	if err := exportHTML(source, "doc.md", absPath, outDir); err != nil {
		t.Fatalf("exportHTML: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "doc.html"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	html := string(data)

	structural := []string{
		"<!DOCTYPE html>",
		`<html lang="en">`,
		"<head>",
		`<meta charset="utf-8">`,
		"<title>doc</title>",
		"<style>",
		"</style>",
		"</head>",
		"<body>",
		`<article class="markdown-body">`,
		"</article>",
		"</body>",
		"</html>",
	}
	for _, s := range structural {
		if !strings.Contains(html, s) {
			t.Errorf("missing structural element: %q", s)
		}
	}

	// Verify CSS is embedded (both export CSS and chroma CSS)
	if !strings.Contains(html, ".markdown-body") {
		t.Error("export CSS not embedded")
	}
	if !strings.Contains(html, ".chroma") {
		t.Error("chroma CSS not embedded")
	}
}

func TestExportHTML_CodeHighlighting(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	source := []byte("# Code\n\n```go\npackage main\n\nfunc main() {}\n```\n")
	if err := os.WriteFile(filepath.Join(srcDir, "code.md"), source, 0o644); err != nil {
		t.Fatal(err)
	}

	absPath := filepath.Join(srcDir, "code.md")
	if err := exportHTML(source, "code.md", absPath, outDir); err != nil {
		t.Fatalf("exportHTML: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(outDir, "code.html"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	html := string(data)
	// Chroma wraps tokens in spans with class attributes
	if !strings.Contains(html, "chroma") {
		t.Error("output should contain chroma-highlighted code")
	}
}

func TestHighlightCodeBlocks(t *testing.T) {
	source := []byte("# Demo\n\n```python\nprint('hello')\n```\n")
	md := newGoldmark()
	result := highlightCodeBlocks(source, md)

	if result == "" {
		t.Fatal("highlightCodeBlocks returned empty string")
	}
	if !strings.Contains(result, "chroma") {
		t.Error("result should contain chroma class")
	}
	// The original <code class="language-python"> should be replaced
	if strings.Contains(result, `class="language-python"`) {
		t.Error("original language-python code block was not replaced")
	}
}

func TestHighlightCodeBlocks_NoCodeBlocks(t *testing.T) {
	source := []byte("# Just text\n\nNo code here.\n")
	md := newGoldmark()
	result := highlightCodeBlocks(source, md)

	// Should still return the rendered HTML (just no chroma replacements)
	if result == "" {
		t.Fatal("expected non-empty result for plain markdown")
	}
	if !strings.Contains(result, "Just text") {
		t.Error("result should contain the heading text")
	}
}

func TestChromaCSS(t *testing.T) {
	css := chromaCSS()
	if css == "" {
		t.Fatal("chromaCSS returned empty string")
	}
	if !strings.Contains(css, ".chroma") {
		t.Error("CSS should contain .chroma selectors")
	}
}

func TestCopyReferencedImages_Local(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Create a subdirectory with an image
	imgDir := filepath.Join(srcDir, "images")
	if err := os.MkdirAll(imgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	imgData := []byte("fake-png-data")
	if err := os.WriteFile(filepath.Join(imgDir, "photo.png"), imgData, 0o644); err != nil {
		t.Fatal(err)
	}

	source := []byte("![alt text](images/photo.png)\n")
	copyReferencedImages(source, srcDir, outDir)

	copied, err := os.ReadFile(filepath.Join(outDir, "images", "photo.png"))
	if err != nil {
		t.Fatalf("image not copied: %v", err)
	}
	if string(copied) != string(imgData) {
		t.Error("copied image content mismatch")
	}
}

func TestCopyReferencedImages_SkipRemote(t *testing.T) {
	outDir := t.TempDir()
	srcDir := t.TempDir()

	source := []byte("![remote](https://example.com/img.png)\n![also](http://example.com/other.jpg)\n")
	copyReferencedImages(source, srcDir, outDir)

	// outDir should remain empty (no files created for remote URLs)
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files in outDir for remote URLs, got %d", len(entries))
	}
}

func TestCopyReferencedImages_PathTraversal(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Create a file outside srcDir that a traversal would try to read
	parentDir := filepath.Dir(srcDir)
	secretFile := filepath.Join(parentDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("sensitive"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(secretFile)

	// Attempt path traversal via ../
	source := []byte("![hack](../secret.txt)\n")
	copyReferencedImages(source, srcDir, outDir)

	// outDir should be completely empty — traversal blocked on both src and dst
	entries, _ := os.ReadDir(outDir)
	if len(entries) != 0 {
		t.Errorf("path traversal was NOT blocked: %d files written to outDir", len(entries))
	}
}

func TestCopyReferencedImages_PathTraversal_EtcPasswd(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	source := []byte("![](../../etc/passwd)\n")
	copyReferencedImages(source, srcDir, outDir)

	// Verify nothing was created in outDir
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("path traversal attempt created files in outDir: %d entries", len(entries))
	}
}

func TestDetectPDFTool(t *testing.T) {
	// Can't guarantee any PDF tool is installed, just verify no panic
	tool, args, err := detectPDFTool()
	if err != nil {
		t.Logf("no PDF tool found (expected in CI): %v", err)
		return
	}
	t.Logf("detected PDF tool: %s %v", tool, args)
	if tool == "" {
		t.Error("tool path should be non-empty when err is nil")
	}
}

func TestImagePattern(t *testing.T) {
	tests := []struct {
		input   string
		wantAlt string
		wantSrc string
	}{
		{"![alt](image.png)", "alt", "image.png"},
		{"![](path/to/img.jpg)", "", "path/to/img.jpg"},
		{"![screenshot](../docs/screen.gif)", "screenshot", "../docs/screen.gif"},
		{"![logo](https://example.com/logo.svg)", "logo", "https://example.com/logo.svg"},
		{"![a b c](file with spaces.png)", "a b c", "file with spaces.png"},
	}
	for _, tt := range tests {
		matches := imagePattern.FindStringSubmatch(tt.input)
		if matches == nil {
			t.Errorf("imagePattern did not match %q", tt.input)
			continue
		}
		if matches[1] != tt.wantAlt {
			t.Errorf("imagePattern(%q) alt = %q, want %q", tt.input, matches[1], tt.wantAlt)
		}
		if matches[2] != tt.wantSrc {
			t.Errorf("imagePattern(%q) src = %q, want %q", tt.input, matches[2], tt.wantSrc)
		}
	}
}

func TestImagePattern_NoMatch(t *testing.T) {
	noMatch := []string{
		"[not an image](link.md)",
		"plain text",
		"![unclosed](path",
	}
	for _, s := range noMatch {
		if imagePattern.MatchString(s) {
			t.Errorf("imagePattern should NOT match %q", s)
		}
	}
}

func TestExport_EmptyIndex(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	// Empty directory, no markdown files
	idx := buildIndex(t, srcDir)
	err := Export(idx, Options{Format: FormatHTML, OutputDir: outDir})
	if err == nil {
		t.Fatal("expected error for empty index")
	}
	if !strings.Contains(err.Error(), "no markdown files") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestExport_Progress(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	files := []string{"a.md", "b.md", "c.md"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	idx := buildIndex(t, srcDir)

	var calls []struct {
		current, total int
		file           string
	}

	err := Export(idx, Options{
		Format:    FormatHTML,
		OutputDir: outDir,
		Progress: func(current, total int, file string) {
			calls = append(calls, struct {
				current, total int
				file           string
			}{current, total, file})
		},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if len(calls) != len(files) {
		t.Fatalf("progress called %d times, want %d", len(calls), len(files))
	}

	for i, c := range calls {
		if c.current != i+1 {
			t.Errorf("call %d: current = %d, want %d", i, c.current, i+1)
		}
		if c.total != len(files) {
			t.Errorf("call %d: total = %d, want %d", i, c.total, len(files))
		}
		if c.file == "" {
			t.Errorf("call %d: file is empty", i)
		}
	}
}

func TestExport_SpecificFiles(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	for _, name := range []string{"include.md", "exclude.md"} {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte("# "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	idx := buildIndex(t, srcDir)
	err := Export(idx, Options{
		Format:    FormatHTML,
		OutputDir: outDir,
		Files:     []string{"include.md"},
	})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "include.html")); err != nil {
		t.Error("include.html should exist")
	}
	if _, err := os.Stat(filepath.Join(outDir, "exclude.html")); err == nil {
		t.Error("exclude.html should NOT exist")
	}
}

func TestExport_SubdirectoryFiles(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	subDir := filepath.Join(srcDir, "docs")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "nested.md"), []byte("# Nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx := buildIndex(t, srcDir)
	if err := Export(idx, Options{Format: FormatHTML, OutputDir: outDir}); err != nil {
		t.Fatalf("Export: %v", err)
	}

	outFile := filepath.Join(outDir, "docs", "nested.html")
	if _, err := os.Stat(outFile); err != nil {
		t.Errorf("nested output file not created at %s: %v", outFile, err)
	}
}

func TestExportFile_UnknownFormat(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "test.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	idx := buildIndex(t, srcDir)
	err := exportFile(idx, "test.md", outDir, Format(99))
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExportFile_NotInIndex(t *testing.T) {
	srcDir := t.TempDir()
	outDir := t.TempDir()

	idx := buildIndex(t, srcDir)
	err := exportFile(idx, "nonexistent.md", outDir, FormatHTML)
	if err == nil {
		t.Fatal("expected error for file not in index")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestExport_DefaultOutputDir(t *testing.T) {
	srcDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "test.md"), []byte("# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Change to a temp dir so the default "fur-export" dir is created there
	origDir, _ := os.Getwd()
	tmpWork := t.TempDir()
	os.Chdir(tmpWork)
	defer os.Chdir(origDir)

	idx := buildIndex(t, srcDir)
	err := Export(idx, Options{Format: FormatHTML})
	if err != nil {
		t.Fatalf("Export: %v", err)
	}

	defaultOut := filepath.Join(tmpWork, "fur-export", "test.html")
	if _, err := os.Stat(defaultOut); err != nil {
		t.Errorf("default output dir not used: %v", err)
	}
}

// newGoldmark creates a goldmark instance matching what exportHTML uses.
func newGoldmark() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
}

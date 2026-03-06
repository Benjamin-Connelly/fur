package export

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldhtml "github.com/yuin/goldmark/renderer/html"

	"github.com/Benjamin-Connelly/lookit/internal/index"
)

// Format identifies the export output format.
type Format int

const (
	FormatHTML Format = iota
	FormatPDF
)

// ProgressFunc is called with (current, total, filename) during export.
type ProgressFunc func(current, total int, file string)

// Options configures the export operation.
type Options struct {
	Format   Format
	OutputDir string
	Files    []string // specific files, or empty for all markdown
	Progress ProgressFunc
}

// Export converts markdown files to the specified output format.
func Export(idx *index.Index, opts Options) error {
	files := opts.Files
	if len(files) == 0 {
		for _, e := range idx.MarkdownFiles() {
			files = append(files, e.RelPath)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no markdown files found")
	}

	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "lookit-export"
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("creating output dir: %w", err)
	}

	for i, file := range files {
		if opts.Progress != nil {
			opts.Progress(i+1, len(files), file)
		}
		if err := exportFile(idx, file, outputDir, opts.Format); err != nil {
			return fmt.Errorf("exporting %s: %w", file, err)
		}
	}

	return nil
}

func exportFile(idx *index.Index, relPath, outputDir string, format Format) error {
	entry := idx.Lookup(relPath)
	if entry == nil {
		return fmt.Errorf("file not found in index: %s", relPath)
	}

	source, err := os.ReadFile(entry.Path)
	if err != nil {
		return err
	}

	switch format {
	case FormatHTML:
		return exportHTML(source, relPath, entry.Path, outputDir)
	case FormatPDF:
		return fmt.Errorf("PDF export not yet implemented")
	default:
		return fmt.Errorf("unknown format: %d", format)
	}
}

func exportHTML(source []byte, relPath, absPath, outputDir string) error {
	// Render markdown to HTML
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
		goldmark.WithRendererOptions(goldhtml.WithUnsafe()),
	)

	var body bytes.Buffer
	if err := md.Convert(source, &body); err != nil {
		return fmt.Errorf("rendering markdown: %w", err)
	}

	// Apply syntax highlighting to code blocks
	highlighted := highlightCodeBlocks(source, md)

	// Copy referenced images
	srcDir := filepath.Dir(absPath)
	outSubDir := filepath.Join(outputDir, filepath.Dir(relPath))
	if err := os.MkdirAll(outSubDir, 0o755); err != nil {
		return fmt.Errorf("creating output subdir: %w", err)
	}
	copyReferencedImages(source, srcDir, outSubDir)

	// Build complete HTML document
	title := titleFromPath(relPath)
	var doc bytes.Buffer
	doc.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	doc.WriteString("<meta charset=\"utf-8\">\n")
	doc.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&doc, "<title>%s</title>\n", title)
	doc.WriteString("<style>\n")
	doc.WriteString(exportCSS)
	doc.WriteString(chromaCSS())
	doc.WriteString("</style>\n")
	doc.WriteString("</head>\n<body>\n")
	doc.WriteString("<article class=\"markdown-body\">\n")
	if highlighted != "" {
		doc.WriteString(highlighted)
	} else {
		doc.Write(body.Bytes())
	}
	doc.WriteString("\n</article>\n</body>\n</html>\n")

	outName := replaceExt(filepath.Base(relPath), ".html")
	outPath := filepath.Join(outSubDir, outName)
	return os.WriteFile(outPath, doc.Bytes(), 0o644)
}

// highlightCodeBlocks re-renders markdown with chroma syntax highlighting.
func highlightCodeBlocks(source []byte, md goldmark.Markdown) string {
	// First render normally
	var tmp bytes.Buffer
	if err := md.Convert(source, &tmp); err != nil {
		return ""
	}

	content := tmp.String()
	var result bytes.Buffer
	// Replace <pre><code class="language-XXX">...</code></pre> with chroma output
	re := regexp.MustCompile(`(?s)<pre><code class="language-([^"]+)">(.+?)</code></pre>`)
	lastIdx := 0
	for _, loc := range re.FindAllStringSubmatchIndex(content, -1) {
		result.WriteString(content[lastIdx:loc[0]])
		lang := content[loc[2]:loc[3]]
		code := content[loc[4]:loc[5]]
		code = unescapeHTML(code)

		highlighted, err := highlightCode(code, lang)
		if err != nil {
			result.WriteString(content[loc[0]:loc[1]])
		} else {
			result.WriteString(highlighted)
		}
		lastIdx = loc[1]
	}
	result.WriteString(content[lastIdx:])

	if result.Len() == 0 {
		return ""
	}
	return result.String()
}

func highlightCode(code, lang string) (string, error) {
	lexer := lexers.Get(lang)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	formatter := chromahtml.New(
		chromahtml.WithClasses(true),
		chromahtml.PreventSurroundingPre(false),
	)

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, styles.Get("monokai"), iterator); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func chromaCSS() string {
	formatter := chromahtml.New(chromahtml.WithClasses(true))
	var buf bytes.Buffer
	if err := formatter.WriteCSS(&buf, styles.Get("monokai")); err != nil {
		return ""
	}
	return buf.String()
}

func unescapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&#34;", "\"")
	return s
}

var imagePattern = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

func copyReferencedImages(source []byte, srcDir, outDir string) {
	matches := imagePattern.FindAllSubmatch(source, -1)
	for _, m := range matches {
		imgPath := string(m[2])
		// Skip remote URLs
		if strings.HasPrefix(imgPath, "http://") || strings.HasPrefix(imgPath, "https://") {
			continue
		}
		srcPath := filepath.Join(srcDir, imgPath)
		dstPath := filepath.Join(outDir, imgPath)

		// Ensure destination directory exists
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			continue
		}

		data, err := os.ReadFile(srcPath)
		if err != nil {
			continue
		}
		_ = os.WriteFile(dstPath, data, 0o644)
	}
}

func titleFromPath(relPath string) string {
	name := filepath.Base(relPath)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return strings.ReplaceAll(name, "-", " ")
}

func replaceExt(name, newExt string) string {
	ext := filepath.Ext(name)
	return name[:len(name)-len(ext)] + newExt
}

const exportCSS = `
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Helvetica, Arial, sans-serif;
  line-height: 1.6;
  color: #24292e;
  background: #fff;
  max-width: 800px;
  margin: 0 auto;
  padding: 2rem 1rem;
}
.markdown-body h1, .markdown-body h2, .markdown-body h3,
.markdown-body h4, .markdown-body h5, .markdown-body h6 {
  margin-top: 1.5em;
  margin-bottom: 0.5em;
  font-weight: 600;
  line-height: 1.25;
}
.markdown-body h1 { font-size: 2em; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
.markdown-body h2 { font-size: 1.5em; border-bottom: 1px solid #eaecef; padding-bottom: 0.3em; }
.markdown-body p { margin-bottom: 1em; }
.markdown-body code {
  background: #f6f8fa;
  padding: 0.2em 0.4em;
  border-radius: 3px;
  font-size: 85%;
}
.markdown-body pre {
  background: #272822;
  color: #f8f8f2;
  padding: 1em;
  border-radius: 6px;
  overflow-x: auto;
  margin-bottom: 1em;
}
.markdown-body pre code {
  background: none;
  padding: 0;
  font-size: 85%;
  color: inherit;
}
.markdown-body blockquote {
  border-left: 4px solid #dfe2e5;
  padding: 0 1em;
  color: #6a737d;
  margin-bottom: 1em;
}
.markdown-body ul, .markdown-body ol { padding-left: 2em; margin-bottom: 1em; }
.markdown-body li { margin-bottom: 0.25em; }
.markdown-body table { border-collapse: collapse; width: 100%; margin-bottom: 1em; }
.markdown-body th, .markdown-body td {
  border: 1px solid #dfe2e5;
  padding: 6px 13px;
}
.markdown-body th { background: #f6f8fa; font-weight: 600; }
.markdown-body img { max-width: 100%; }
.markdown-body a { color: #0366d6; text-decoration: none; }
.markdown-body a:hover { text-decoration: underline; }
.markdown-body hr { border: none; border-top: 1px solid #eaecef; margin: 1.5em 0; }
`

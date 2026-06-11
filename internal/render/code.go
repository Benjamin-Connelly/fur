package render

import (
	"bytes"
	"io"
	"path/filepath"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	htmlfmt "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/spf13/afero"

	"github.com/Benjamin-Connelly/fur/internal/theme"
)

// CodeRenderer provides syntax highlighting via Chroma.
type CodeRenderer struct {
	theme    string
	terminal bool // true for TUI output, false for HTML
	fs       afero.Fs
}

// NewCodeRenderer creates a code renderer.
// Set terminal=true for TUI (terminal256) output, false for HTML output.
func NewCodeRenderer(theme string, terminal bool) *CodeRenderer {
	return &CodeRenderer{
		theme:    theme,
		terminal: terminal,
		fs:       afero.NewOsFs(),
	}
}

// SetFs sets the filesystem for file operations.
func (r *CodeRenderer) SetFs(fs afero.Fs) {
	r.fs = fs
}

// SetTheme changes the highlighting theme at runtime.
func (r *CodeRenderer) SetTheme(theme string) {
	r.theme = theme
}

// Highlight returns syntax-highlighted content for the given file.
func (r *CodeRenderer) Highlight(filename, source string) (string, error) {
	lexer := r.detectLexer(filename, source)
	style := r.getStyle()
	formatter := r.getFormatter()

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source, nil
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return source, nil
	}

	return buf.String(), nil
}

// HighlightFile reads a file and returns syntax-highlighted content.
func (r *CodeRenderer) HighlightFile(filePath string) (string, error) {
	data, err := afero.ReadFile(r.fs, filePath)
	if err != nil {
		return "", err
	}
	return r.Highlight(filepath.Base(filePath), string(data))
}

// HighlightLines highlights source with a specific line range emphasized.
// Lines outside startLine..endLine are rendered normally; lines within the
// range receive Chroma's highlight style.
func (r *CodeRenderer) HighlightLines(filename, source string, startLine, endLine int) (string, error) {
	lexer := r.detectLexer(filename, source)
	style := r.getStyle()

	var formatter chroma.Formatter
	if r.terminal {
		// Terminal formatter doesn't support line highlighting; fall back to normal
		formatter = r.getFormatter()
	} else {
		formatter = htmlfmt.New(
			htmlfmt.WithClasses(true),
			htmlfmt.WithLineNumbers(true),
			htmlfmt.HighlightLines([][2]int{{startLine, endLine}}),
		)
	}

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return source, nil
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return source, nil
	}

	return buf.String(), nil
}

// HighlightToWriter writes syntax-highlighted content to a writer.
func (r *CodeRenderer) HighlightToWriter(w io.Writer, filename, source string) error {
	lexer := r.detectLexer(filename, source)
	style := r.getStyle()
	formatter := r.getFormatter()

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		_, writeErr := w.Write([]byte(source))
		return writeErr
	}

	return formatter.Format(w, style, iterator)
}

// GetLanguage detects the language for a filename.
// Returns an empty string if no language is detected.
func (r *CodeRenderer) GetLanguage(filename string) string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		return ""
	}
	cfg := lexer.Config()
	if cfg == nil {
		return ""
	}
	return cfg.Name
}

// ListLanguages returns all supported language names.
func (r *CodeRenderer) ListLanguages() []string {
	return lexers.Names(false)
}

// CSS returns the Chroma CSS classes for the current theme (HTML mode only).
func (r *CodeRenderer) CSS() (string, error) {
	style := r.getStyle()
	formatter := htmlfmt.New(htmlfmt.WithClasses(true))

	var buf bytes.Buffer
	if err := formatter.WriteCSS(&buf, style); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (r *CodeRenderer) detectLexer(filename, source string) chroma.Lexer {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(source)
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}
	return chroma.Coalesce(lexer)
}

func (r *CodeRenderer) getStyle() *chroma.Style {
	name := theme.Resolve(r.theme).Chroma
	if name == "" {
		return styles.Fallback
	}
	style := styles.Get(name)
	if style == nil {
		style = styles.Fallback
	}
	return style
}

func (r *CodeRenderer) getFormatter() chroma.Formatter {
	if r.terminal {
		return formatters.Get("terminal256")
	}
	return htmlfmt.New(
		htmlfmt.WithClasses(true),
		htmlfmt.WithLineNumbers(true),
	)
}

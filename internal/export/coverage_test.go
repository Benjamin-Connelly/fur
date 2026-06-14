package export

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/index"
)

func buildExportIdx(t *testing.T) *index.Index {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc.md"), []byte("# Doc\n\nbody\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	return idx
}

// TestExportPDFNoTool drives the PDF path with no converter on PATH, exercising
// the Export loop, the Progress callback, exportFile's PDF dispatch, and
// detectPDFTool/exportPDF's error branch.
func TestExportPDFNoTool(t *testing.T) {
	t.Setenv("PATH", "") // no chromium/chrome/wkhtmltopdf discoverable
	idx := buildExportIdx(t)

	var progressCalls int
	err := Export(idx, Options{
		Format:    FormatPDF,
		OutputDir: filepath.Join(t.TempDir(), "out"),
		Progress:  func(cur, total int, file string) { progressCalls++ },
	})
	if err == nil {
		t.Fatal("expected error when no PDF tool is available")
	}
	if progressCalls == 0 {
		t.Error("Progress callback was never invoked")
	}
}

// TestExportNoMarkdown covers the empty-set error branch.
func TestExportNoMarkdown(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package x\n"), 0o644)
	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatal(err)
	}
	if err := Export(idx, Options{Format: FormatHTML, OutputDir: t.TempDir()}); err == nil {
		t.Error("expected 'no markdown files' error")
	}
}

// TestExportUnknownFormat covers the default switch branch in exportFile.
func TestExportUnknownFormat(t *testing.T) {
	idx := buildExportIdx(t)
	if err := Export(idx, Options{Format: Format(99), OutputDir: t.TempDir()}); err == nil {
		t.Error("expected error for unknown format")
	}
}

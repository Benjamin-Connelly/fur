package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/index"
)

// TestFileListSanitizesAnsiFilename is the Chain J regression guard.
//
// A directory adversary controls the names of files in the tree a victim
// browses. A name carrying an OSC sequence — e.g.
// "ev\x1b]0;PWNED\x07il.md" sets the terminal window title — would, before
// the fix, be written verbatim into the file tree and reprogram the victim's
// terminal. The renderer must pass every user-controlled name through
// sanitize.Terminal.
//
// lipgloss legitimately emits CSI (ESC[) color codes, but never OSC (ESC])
// or BEL (0x07), so this test keys on those: the OSC introducer, the BEL
// terminator, and the smuggled "PWNED" payload must all be absent.
// References lookit-9py.3.14 / .4.3.
func TestFileListSanitizesAnsiFilename(t *testing.T) {
	dir := t.TempDir()
	hostile := "ev\x1b]0;PWNED\x07il.md"
	if err := os.WriteFile(filepath.Join(dir, hostile), []byte("# x\n"), 0o644); err != nil {
		t.Skipf("filesystem rejected control-char filename: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plain.md"), []byte("# y\n"), 0o644); err != nil {
		t.Fatalf("write plain: %v", err)
	}

	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}

	m := NewFileListModel(idx)
	m.height = 30

	assertNoTerminalSmuggling(t, "tree view", m.viewTree())

	// Filtered view renders RelPath through a different path.
	m.SetFilter("il")
	assertNoTerminalSmuggling(t, "filtered view", m.viewFiltered())
}

func assertNoTerminalSmuggling(t *testing.T, label, out string) {
	t.Helper()
	if strings.Contains(out, "\x07") {
		t.Errorf("%s: rendered output contains a BEL byte — a filename's OSC "+
			"sequence reached the terminal (Chain J)", label)
	}
	if strings.Contains(out, "\x1b]") {
		t.Errorf("%s: rendered output contains an OSC introducer (ESC]) from a "+
			"filename (Chain J)", label)
	}
	if strings.Contains(out, "PWNED") {
		t.Errorf("%s: rendered output contains the smuggled OSC title payload "+
			"(Chain J)", label)
	}
}

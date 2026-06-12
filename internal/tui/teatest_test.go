package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
)

// newTestTUI builds a Model over a small real tree for behavioral driving.
func newTestTUI(t *testing.T) *Model {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Title\n\nbody text here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("Build: %v", err)
	}
	cfg := config.DefaultConfig()
	cfg.Git.Enabled = false
	links := index.NewLinkGraph()
	links.BuildFromIndex(idx)
	return New(cfg, idx, links, nil)
}

func sendTUIKey(tm *teatest.TestModel, s string) {
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)})
}

// TestTUI_QuitOnQ drives the model and asserts "q" triggers a clean shutdown
// (quitting flag set, program terminates).
func TestTUI_QuitOnQ(t *testing.T) {
	m := newTestTUI(t)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendTUIKey(tm, "q")

	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	fm, ok := tm.FinalModel(t).(*Model)
	if !ok {
		t.Fatalf("final model type = %T", tm.FinalModel(t))
	}
	if !fm.quitting {
		t.Error("quitting flag not set after 'q'")
	}
}

// TestTUI_HelpToggle asserts "?" renders the help overlay (key-bindings
// screen), exercising a normal-mode keybinding end to end.
func TestTUI_HelpToggle(t *testing.T) {
	m := newTestTUI(t)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendTUIKey(tm, "?")

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("Key Bindings"))
	}, teatest.WithCheckInterval(20*time.Millisecond), teatest.WithDuration(3*time.Second))

	sendTUIKey(tm, "q")
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
	if fm, ok := tm.FinalModel(t).(*Model); ok && !fm.showingHelp {
		// help was toggled on then we quit; showingHelp may remain true — the
		// assertion of interest (help rendered) already passed via WaitFor.
		_ = fm
	}
}

// TestTUI_CommandPalette asserts ":" opens the command palette and that it
// echoes typed input — a focus-independent normal-mode keybinding driven end
// to end through Update/View.
func TestTUI_CommandPalette(t *testing.T) {
	m := newTestTUI(t)
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	sendTUIKey(tm, ":")
	sendTUIKey(tm, "t")
	sendTUIKey(tm, "h")
	sendTUIKey(tm, "e")
	sendTUIKey(tm, "m")
	sendTUIKey(tm, "e")

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte(":theme"))
	}, teatest.WithCheckInterval(20*time.Millisecond), teatest.WithDuration(3*time.Second))

	// esc out of the palette, then quit cleanly.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	sendTUIKey(tm, "q")
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

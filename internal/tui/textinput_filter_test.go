package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestFilterTextinputEditing verifies the filter field gained real in-place
// editing via bubbles/textinput (lookit-6gy): typing appends, and the cursor
// can move left to insert in the middle — behavior the old append-only string
// could not do. Driven directly through Update for determinism.
func TestFilterTextinputEditing(t *testing.T) {
	m := testModel(t)
	m.focus = PanelFileList

	m, _ = sendKey(m, "/")
	if m.mode != modeFilter {
		t.Fatalf("expected filter mode, got %v", m.mode)
	}

	// Type "ac".
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "c")
	if got := m.fileList.filterInput.Value(); got != "ac" {
		t.Fatalf("after typing 'ac', value = %q", got)
	}

	// Move the cursor left one (between a and c) and insert "b" → "abc".
	m, _ = sendSpecialKey(m, tea.KeyLeft)
	m, _ = sendKey(m, "b")
	if got := m.fileList.filterInput.Value(); got != "abc" {
		t.Errorf("after left+insert 'b', value = %q, want \"abc\" (in-place edit)", got)
	}
	if m.fileList.filter != "abc" {
		t.Errorf("m.filter not synced with textinput: %q", m.fileList.filter)
	}

	// Backspace deletes before the cursor (now after the inserted b) → "ac".
	m, _ = sendSpecialKey(m, tea.KeyBackspace)
	if got := m.fileList.filterInput.Value(); got != "ac" {
		t.Errorf("after backspace, value = %q, want \"ac\"", got)
	}

	// esc clears and exits filter mode.
	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.mode != modeNormal {
		t.Errorf("esc should exit filter mode, mode = %v", m.mode)
	}
	if m.fileList.filtering {
		t.Error("esc should stop filtering")
	}
}

// TestFilterTextinputRendersInput confirms the filtered view renders the
// textinput (so the user sees the prompt and typed text with a cursor).
func TestFilterTextinputRendersInput(t *testing.T) {
	m := testModel(t)
	m.focus = PanelFileList
	m, _ = sendKey(m, "/")
	m, _ = sendKey(m, "x")

	view := m.fileList.viewFiltered()
	if !strings.Contains(view, "/ ") || !strings.Contains(view, "x") {
		t.Errorf("filtered view missing prompt or typed text:\n%s", view)
	}
}

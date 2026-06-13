package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// editInPlace types "ac", moves the cursor left one, and inserts "b" — an edit
// the old append-only inputs could not do. Returns nothing; callers assert the
// resulting value.
func editInPlace(t *testing.T, m *Model) *Model {
	t.Helper()
	m, _ = sendKey(m, "a")
	m, _ = sendKey(m, "c")
	m, _ = sendSpecialKey(m, tea.KeyLeft)
	m, _ = sendKey(m, "b")
	return m
}

// TestSearchTextinputEditing — preview search field gains in-place editing.
func TestSearchTextinputEditing(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview
	m.preview.SetContent("doc.md", "abc def\nabcb line\n")

	m, _ = sendKey(m, "/")
	if m.mode != modeSearch {
		t.Fatalf("expected search mode, got %v", m.mode)
	}
	m = editInPlace(t, m)
	if got := m.preview.searchInput.Value(); got != "abc" {
		t.Errorf("search value = %q, want \"abc\" (in-place edit)", got)
	}
	if m.preview.searchQuery != "abc" {
		t.Errorf("searchQuery not synced: %q", m.preview.searchQuery)
	}
	// esc exits search.
	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.mode != modeNormal {
		t.Errorf("esc should exit search, mode=%v", m.mode)
	}
}

// TestCommandPaletteTextinputEditing — command palette gains in-place editing
// and refilters on each change.
func TestCommandPaletteTextinputEditing(t *testing.T) {
	m := testModel(t)

	m, _ = sendKey(m, ":")
	if m.mode != modeCommand {
		t.Fatalf("expected command mode, got %v", m.mode)
	}
	m = editInPlace(t, m)
	if got := m.cmdPalette.ti.Value(); got != "abc" {
		t.Errorf("palette value = %q, want \"abc\"", got)
	}
	if m.cmdPalette.input != "abc" {
		t.Errorf("palette input not synced: %q", m.cmdPalette.input)
	}
	if !strings.Contains(m.cmdPalette.View(), ":abc") {
		t.Errorf("palette view missing :abc:\n%s", m.cmdPalette.View())
	}
}

// TestHeadingJumpTextinputEditing — heading-jump query gains in-place editing.
func TestHeadingJumpTextinputEditing(t *testing.T) {
	m := testModel(t)

	m, _ = sendSpecialKey(m, tea.KeyCtrlG)
	if m.mode != modeHeadingJump {
		t.Fatalf("expected heading-jump mode, got %v", m.mode)
	}
	m = editInPlace(t, m)
	if got := m.headingJumpTI.Value(); got != "abc" {
		t.Errorf("heading value = %q, want \"abc\"", got)
	}
	if m.headingJumpInput != "abc" {
		t.Errorf("headingJumpInput not synced: %q", m.headingJumpInput)
	}
}

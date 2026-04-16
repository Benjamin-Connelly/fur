package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benjamin-Connelly/fur/internal/config"
	"github.com/Benjamin-Connelly/fur/internal/index"
)

// testModel creates a Model with a minimal temp directory containing a markdown
// file and a code file. Returns the model and cleanup is handled by t.TempDir.
func testModel(t *testing.T) *Model {
	t.Helper()
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello\n\nWorld.\n\n## Section\n\nMore text.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0o755)
	os.WriteFile(filepath.Join(dir, "docs", "guide.md"), []byte("# Guide\n\nSee [README](../README.md).\n"), 0o644)

	cfg := config.DefaultConfig()
	cfg.Root = dir
	cfg.Theme = "dark"

	idx := index.New(dir)
	if err := idx.Build(); err != nil {
		t.Fatalf("index.Build: %v", err)
	}

	links := index.NewLinkGraph()
	links.SetLinks("docs/guide.md", []index.Link{
		{Source: "docs/guide.md", Target: "README.md", Text: "README"},
	})

	m := New(cfg, idx, links, nil)
	// Set a reasonable terminal size
	m.width = 120
	m.height = 40
	m.recalcLayout()
	return m
}

// sendKey sends a key message through Update and returns the result.
func sendKey(m *Model, key string) (*Model, tea.Cmd) {
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	return result.(*Model), cmd
}

// sendSpecialKey sends a special key (enter, esc, tab, etc.) through Update.
func sendSpecialKey(m *Model, keyType tea.KeyType) (*Model, tea.Cmd) {
	result, cmd := m.Update(tea.KeyMsg{Type: keyType})
	return result.(*Model), cmd
}

// sendMsg sends a tea.Msg through Update.
func sendMsg(m *Model, msg tea.Msg) (*Model, tea.Cmd) {
	result, cmd := m.Update(msg)
	return result.(*Model), cmd
}

func TestModel_New(t *testing.T) {
	m := testModel(t)
	if m == nil {
		t.Fatal("New() should not return nil")
	}
	if m.focus != PanelFileList {
		t.Error("initial focus should be PanelFileList")
	}
	if m.quitting {
		t.Error("should not start in quitting state")
	}
	if m.searchMode != "filename" {
		t.Errorf("default searchMode should be 'filename', got %q", m.searchMode)
	}
}

func TestModel_Init(t *testing.T) {
	m := testModel(t)
	cmd := m.Init()
	// Without mouse enabled, Init returns nil
	if cmd != nil {
		t.Error("Init should return nil without mouse enabled")
	}
}

func TestModel_Init_WithMouse(t *testing.T) {
	m := testModel(t)
	m.cfg.Mouse = true
	cmd := m.Init()
	if cmd == nil {
		t.Error("Init should return EnableMouseCellMotion with mouse enabled")
	}
}

func TestModel_Quit(t *testing.T) {
	m := testModel(t)
	m, _ = sendKey(m, "q")
	if !m.quitting {
		t.Error("should be quitting after 'q'")
	}
}

func TestModel_TabFocusSwitching(t *testing.T) {
	m := testModel(t)
	if m.focus != PanelFileList {
		t.Fatal("should start at PanelFileList")
	}

	// Tab: FileList -> Preview
	m, _ = sendSpecialKey(m, tea.KeyTab)
	if m.focus != PanelPreview {
		t.Errorf("Tab should switch to Preview, got %d", m.focus)
	}

	// Tab: Preview -> FileList (no side panel visible)
	m, _ = sendSpecialKey(m, tea.KeyTab)
	if m.focus != PanelFileList {
		t.Errorf("Tab should switch back to FileList, got %d", m.focus)
	}
}

func TestModel_TabWithSidePanel(t *testing.T) {
	m := testModel(t)

	// Open TOC side panel
	m, _ = sendKey(m, "t")
	if !m.sidePanel.Visible() {
		t.Fatal("TOC panel should be visible")
	}

	// Go back to file list
	m.focus = PanelFileList

	// Tab: FileList -> Preview -> Side -> FileList
	m, _ = sendSpecialKey(m, tea.KeyTab)
	if m.focus != PanelPreview {
		t.Errorf("expected Preview, got %d", m.focus)
	}
	m, _ = sendSpecialKey(m, tea.KeyTab)
	if m.focus != PanelSide {
		t.Errorf("expected Side, got %d", m.focus)
	}
	m, _ = sendSpecialKey(m, tea.KeyTab)
	if m.focus != PanelFileList {
		t.Errorf("expected FileList, got %d", m.focus)
	}
}

func TestModel_EscFromPreview(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview

	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.focus != PanelFileList {
		t.Error("Esc from preview should return to file list")
	}
}

func TestModel_EscFromSidePanel(t *testing.T) {
	m := testModel(t)
	m.sidePanel.Toggle(PanelTOC)
	m.focus = PanelSide

	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.focus != PanelPreview {
		t.Error("Esc from side panel should return to preview")
	}
	if m.sidePanel.Visible() {
		t.Error("side panel should close on Esc")
	}
}

func TestModel_SearchModeEntry(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview

	m, _ = sendKey(m, "/")
	if !m.preview.searchMode {
		t.Error("/ in preview should enter search mode")
	}
}

func TestModel_FilterModeEntry(t *testing.T) {
	m := testModel(t)
	// From file list, / enters filter mode
	m, _ = sendKey(m, "/")
	if !m.fileList.filtering {
		t.Error("/ from file list should enter filter mode")
	}
	if m.focus != PanelFileList {
		t.Error("focus should stay on file list during filter")
	}
}

func TestModel_CommandPaletteEntry(t *testing.T) {
	m := testModel(t)
	m, _ = sendKey(m, ":")
	if !m.cmdPalette.IsActive() {
		t.Error(": should open command palette")
	}
}

func TestModel_CommandPalette_EscClose(t *testing.T) {
	m := testModel(t)
	m, _ = sendKey(m, ":")
	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.cmdPalette.IsActive() {
		t.Error("Esc should close command palette")
	}
}

func TestModel_HelpToggle(t *testing.T) {
	m := testModel(t)
	// Give preview some content first
	m.preview.SetContent("test.md", "initial content")

	m, _ = sendKey(m, "?")
	if !m.showingHelp {
		t.Error("? should toggle help on")
	}
	if m.focus != PanelPreview {
		t.Error("help should focus preview")
	}

	// Toggle off
	m, _ = sendKey(m, "?")
	if m.showingHelp {
		t.Error("? again should toggle help off")
	}
}

func TestModel_WindowResize(t *testing.T) {
	m := testModel(t)
	m, _ = sendMsg(m, tea.WindowSizeMsg{Width: 200, Height: 50})
	if m.width != 200 || m.height != 50 {
		t.Errorf("expected 200x50, got %dx%d", m.width, m.height)
	}
}

func TestModel_PreviewLoaded(t *testing.T) {
	m := testModel(t)
	m, _ = sendMsg(m, PreviewLoadedMsg{Path: "test.md", Content: "# Hello\nWorld"})
	if m.preview.filePath != "test.md" {
		t.Error("preview should load the file")
	}
	if m.focus != PanelPreview {
		t.Error("focus should switch to preview after load")
	}
}

func TestModel_StatusMessage(t *testing.T) {
	m := testModel(t)
	m, cmd := sendMsg(m, StatusMsg{Text: "hello"})
	if m.status.message != "hello" {
		t.Error("status message should be set")
	}
	if cmd == nil {
		t.Error("StatusMsg should return a tick cmd for auto-clear")
	}
}

func TestModel_ClearStatus(t *testing.T) {
	m := testModel(t)
	m.status.SetMessage("temp")
	m, _ = sendMsg(m, clearStatusMsg{})
	if m.status.message != "" {
		t.Error("clearStatusMsg should clear the message")
	}
}

func TestModel_MouseScroll(t *testing.T) {
	m := testModel(t)
	m.cfg.Mouse = true
	m.preview.SetContent("test.md", "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\nline10")
	m.preview.height = 5

	m, _ = sendMsg(m, tea.MouseMsg{Type: tea.MouseWheelDown})
	if m.preview.scroll != 3 {
		t.Errorf("mouse wheel down should scroll by 3, got %d", m.preview.scroll)
	}

	m, _ = sendMsg(m, tea.MouseMsg{Type: tea.MouseWheelUp})
	if m.preview.scroll != 0 {
		t.Errorf("mouse wheel up should scroll back, got %d", m.preview.scroll)
	}
}

func TestModel_MouseDisabled(t *testing.T) {
	m := testModel(t)
	m.cfg.Mouse = false
	m.preview.SetContent("test.md", "line1\nline2\nline3\nline4\nline5")
	m.preview.height = 2

	m, _ = sendMsg(m, tea.MouseMsg{Type: tea.MouseWheelDown})
	if m.preview.scroll != 0 {
		t.Error("mouse should be ignored when disabled")
	}
}

func TestModel_FileListNavigation(t *testing.T) {
	m := testModel(t)
	m.focus = PanelFileList
	initial := m.fileList.cursor

	m, _ = sendKey(m, "j")
	if m.fileList.cursor <= initial {
		t.Error("j should move cursor down in file list")
	}

	m, _ = sendKey(m, "k")
	if m.fileList.cursor != initial {
		t.Error("k should move cursor up in file list")
	}
}

func TestModel_PreviewNavigation(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview
	m.preview.SetContent("test.md", "line1\nline2\nline3\nline4\nline5")
	m.preview.height = 20

	m, _ = sendKey(m, "j")
	if m.preview.cursorLine != 1 {
		t.Errorf("j should move cursor down, got %d", m.preview.cursorLine)
	}

	m, _ = sendKey(m, "k")
	if m.preview.cursorLine != 0 {
		t.Errorf("k should move cursor up, got %d", m.preview.cursorLine)
	}
}

func TestModel_VisualMode(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview
	m.preview.SetContent("test.md", "line1\nline2\nline3\nline4\nline5")
	m.preview.height = 20

	m, _ = sendKey(m, "V")
	if !m.preview.visualMode {
		t.Error("V should enter visual mode")
	}

	m, _ = sendKey(m, "j")
	m, _ = sendKey(m, "j")
	if m.preview.visualEnd != 2 {
		t.Errorf("expected visual end at 2, got %d", m.preview.visualEnd)
	}

	// Esc exits visual mode
	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.preview.visualMode {
		t.Error("Esc should exit visual mode")
	}
}

func TestModel_SidePanelNavigation(t *testing.T) {
	m := testModel(t)
	m.sidePanel.Toggle(PanelTOC)
	m.sidePanel.SetTOC([]TOCEntry{
		{Text: "A", Line: 1},
		{Text: "B", Line: 5},
		{Text: "C", Line: 10},
	})
	m.focus = PanelSide

	m, _ = sendKey(m, "j")
	if m.sidePanel.cursor != 1 {
		t.Error("j should move cursor down in side panel")
	}

	m, _ = sendKey(m, "k")
	if m.sidePanel.cursor != 0 {
		t.Error("k should move cursor up in side panel")
	}
}

func TestModel_MarkSetAndJump(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview
	m.preview.SetContent("test.md", "line1\nline2\nline3\nline4\nline5")
	m.preview.height = 20
	m.preview.cursorLine = 3

	// Set mark 'a'
	m, _ = sendKey(m, "m")
	if !m.pendingMark {
		t.Fatal("m should set pendingMark")
	}
	m, _ = sendKey(m, "a")
	if m.pendingMark {
		t.Error("pendingMark should be cleared")
	}
	mk, ok := m.marks['a']
	if !ok {
		t.Fatal("mark 'a' should be set")
	}
	if mk.Cursor != 3 {
		t.Errorf("mark cursor should be 3, got %d", mk.Cursor)
	}

	// Move cursor
	m.preview.cursorLine = 0

	// Jump to mark 'a'
	m, _ = sendKey(m, "'")
	if !m.pendingJump {
		t.Fatal("' should set pendingJump")
	}
	m, _ = sendKey(m, "a")
	if m.pendingJump {
		t.Error("pendingJump should be cleared")
	}
	if m.preview.cursorLine != 3 {
		t.Errorf("cursor should jump to 3, got %d", m.preview.cursorLine)
	}
}

func TestModel_MarkInvalidRegister(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview
	m.preview.SetContent("test.md", "content")
	m.preview.height = 20

	m, _ = sendKey(m, "m")
	// Send non a-z key
	m, _ = sendKey(m, "1")
	if m.pendingMark {
		t.Error("pendingMark should be cleared on invalid register")
	}
	if len(m.marks) != 0 {
		t.Error("no mark should be set for non a-z key")
	}
}

func TestModel_JumpToUnsetMark(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview
	m.preview.SetContent("test.md", "content")
	m.preview.height = 20

	m, _ = sendKey(m, "'")
	m, cmd := sendKey(m, "z")
	// Should set a status message about unset mark
	if cmd == nil {
		// The status message is set directly, not via cmd
	}
	if m.pendingJump {
		t.Error("pendingJump should be cleared")
	}
}

func TestModel_HeadingJump(t *testing.T) {
	m := testModel(t)
	m, _ = sendMsg(m, tea.KeyMsg{Type: tea.KeyCtrlG})
	if !m.headingJump {
		t.Error("ctrl+g should enter heading jump mode")
	}

	// Esc exits heading jump
	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.headingJump {
		t.Error("Esc should exit heading jump")
	}
}

func TestModel_BackForwardNavigation(t *testing.T) {
	m := testModel(t)

	// Simulate navigation history
	m.navigator.Navigate("a.md", 0)
	m.navigator.Navigate("b.md", 5)

	// Backspace goes back
	m, _ = sendSpecialKey(m, tea.KeyBackspace)
	cur := m.navigator.Current()
	if cur == nil || cur.Path != "a.md" {
		t.Error("backspace should navigate back")
	}
}

func TestModel_ModeString(t *testing.T) {
	m := testModel(t)

	m.focus = PanelFileList
	if mode := m.modeString(); mode != "FILES" {
		t.Errorf("expected FILES, got %q", mode)
	}

	m.focus = PanelPreview
	if mode := m.modeString(); mode != "PREVIEW" {
		t.Errorf("expected PREVIEW, got %q", mode)
	}

	m.focus = PanelSide
	if mode := m.modeString(); mode != "PANEL" {
		t.Errorf("expected PANEL, got %q", mode)
	}
}

func TestModel_FilterKey_Esc(t *testing.T) {
	m := testModel(t)
	m.fileList.StartFilter()
	m.fileList.filter = "test"

	m, _ = sendSpecialKey(m, tea.KeyEsc)
	if m.fileList.filtering {
		t.Error("Esc should exit filter mode")
	}
}

func TestModel_FilterKey_Enter(t *testing.T) {
	m := testModel(t)
	m.fileList.StartFilter()

	// Enter should freeze filter (exit filtering mode but keep filter)
	m, _ = sendSpecialKey(m, tea.KeyEnter)
	if m.fileList.filtering {
		t.Error("Enter should exit active filtering")
	}
}

func TestModel_TOCPanelToggle(t *testing.T) {
	m := testModel(t)

	// t opens TOC
	m, _ = sendKey(m, "t")
	if m.sidePanel.Type() != PanelTOC {
		t.Error("t should open TOC panel")
	}
	if m.focus != PanelSide {
		t.Error("focus should move to side panel")
	}

	// t again closes TOC (when focused on it)
	m, _ = sendKey(m, "t")
	if m.sidePanel.Visible() {
		t.Error("t should toggle TOC off when focused")
	}
	if m.focus != PanelPreview {
		t.Error("focus should return to preview after closing TOC")
	}
}

func TestModel_BacklinksPanelToggle(t *testing.T) {
	m := testModel(t)

	m, _ = sendKey(m, "b")
	if m.sidePanel.Type() != PanelBacklinks {
		t.Error("b should open backlinks panel")
	}

	m, _ = sendKey(m, "b")
	if m.sidePanel.Visible() {
		t.Error("b should toggle backlinks off when focused")
	}
}

func TestModel_BookmarksPanelToggle(t *testing.T) {
	m := testModel(t)

	m, _ = sendKey(m, "M")
	if m.sidePanel.Type() != PanelBookmarks {
		t.Error("M should open bookmarks panel")
	}

	m, _ = sendKey(m, "M")
	if m.sidePanel.Visible() {
		t.Error("M should toggle bookmarks off when focused")
	}
}

func TestModel_UnknownMsg(t *testing.T) {
	m := testModel(t)
	// Send an unrecognized message type — should be no-op
	type fakeMsg struct{}
	m2, cmd := sendMsg(m, fakeMsg{})
	if m2 != m {
		t.Error("unknown message should return same model")
	}
	if cmd != nil {
		t.Error("unknown message should return nil cmd")
	}
}

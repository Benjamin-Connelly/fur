package tui

import "testing"

func TestBuildCSVTable(t *testing.T) {
	rows := [][]string{
		{"name", "age"},
		{"alice", "30"},
		{"bob", "25"},
	}
	tbl, ok := buildCSVTable(rows, 10)
	if !ok {
		t.Fatal("buildCSVTable returned ok=false")
	}
	if len(tbl.Columns()) != 2 {
		t.Errorf("columns = %d, want 2", len(tbl.Columns()))
	}
	if tbl.Columns()[0].Title != "name" {
		t.Errorf("first column title = %q, want name", tbl.Columns()[0].Title)
	}
	if len(tbl.Rows()) != 2 {
		t.Errorf("data rows = %d, want 2", len(tbl.Rows()))
	}
}

func TestBuildCSVTable_SanitizesCells(t *testing.T) {
	rows := [][]string{
		{"h\x1b]0;PWNED\x07dr"},
		{"v\x1b[31mal"},
	}
	tbl, ok := buildCSVTable(rows, 5)
	if !ok {
		t.Fatal("ok=false")
	}
	if got := tbl.Columns()[0].Title; got != "hdr" {
		t.Errorf("header not sanitized: %q", got)
	}
	if got := tbl.Rows()[0][0]; got != "val" {
		t.Errorf("cell not sanitized: %q", got)
	}
}

func TestBuildCSVTable_Empty(t *testing.T) {
	if _, ok := buildCSVTable(nil, 5); ok {
		t.Error("expected ok=false for no rows")
	}
}

// TestCSVPreviewInteractiveTable drives a CSV through the preview: it enters
// table mode and row-navigation keys move the table selection (lookit-qqu).
func TestCSVPreviewInteractiveTable(t *testing.T) {
	m := testModel(t)
	m.focus = PanelPreview

	updated, _ := m.Update(previewWithSourceMsg{
		preview:   PreviewLoadedMsg{Path: "data.csv", Content: "name,age\nalice,30\nbob,25\ncarol,40\n"},
		rawSource: "name,age\nalice,30\nbob,25\ncarol,40\n",
		csvRows:   [][]string{{"name", "age"}, {"alice", "30"}, {"bob", "25"}, {"carol", "40"}},
	})
	m = updated.(*Model)

	if !m.preview.InTableMode() {
		t.Fatal("preview did not enter table mode for a CSV")
	}
	startRow := m.preview.dataTable.Cursor()

	// 'j' moves the table selection down.
	m, _ = sendKey(m, "j")
	if m.preview.dataTable.Cursor() == startRow {
		t.Errorf("'j' did not move the table cursor (still %d)", startRow)
	}

	// View renders the table (header present).
	if view := m.preview.View(); view == "" {
		t.Error("table view is empty")
	}

	// Navigating to a non-CSV file leaves table mode.
	updated, _ = m.Update(previewWithSourceMsg{
		preview:   PreviewLoadedMsg{Path: "note.md", Content: "# hi\n"},
		rawSource: "# hi\n",
	})
	m = updated.(*Model)
	if m.preview.InTableMode() {
		t.Error("table mode should clear when navigating to a non-CSV file")
	}
}

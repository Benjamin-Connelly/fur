package tui

import (
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/table"

	"github.com/Benjamin-Connelly/fur/internal/sanitize"
)

// buildCSVTable turns parsed CSV/TSV rows into an interactive bubbles/table.
// The first row is the header; remaining rows are data. Column widths are
// derived from content (capped) and cells are sanitized of terminal control
// bytes (a CSV is untrusted content). Returns false if there is nothing to
// show.
func buildCSVTable(rows [][]string, height int) (table.Model, bool) {
	if len(rows) == 0 {
		return table.Model{}, false
	}

	cols := 0
	for _, r := range rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	if cols == 0 {
		return table.Model{}, false
	}

	const maxColWidth = 40
	widths := make([]int, cols)
	for _, r := range rows {
		for i, cell := range r {
			if w := utf8.RuneCountInString(sanitize.Terminal(cell)); w > widths[i] {
				widths[i] = w
			}
		}
	}
	for i := range widths {
		if widths[i] < 3 {
			widths[i] = 3
		}
		if widths[i] > maxColWidth {
			widths[i] = maxColWidth
		}
	}

	header := rows[0]
	columns := make([]table.Column, cols)
	for i := 0; i < cols; i++ {
		title := ""
		if i < len(header) {
			title = sanitize.Terminal(header[i])
		}
		columns[i] = table.Column{Title: title, Width: widths[i]}
	}

	trows := make([]table.Row, 0, len(rows)-1)
	for _, r := range rows[1:] {
		cells := make(table.Row, cols)
		for i := 0; i < cols; i++ {
			if i < len(r) {
				cells[i] = sanitize.Terminal(r[i])
			}
		}
		trows = append(trows, cells)
	}

	if height < 3 {
		height = 3
	}
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(trows),
		table.WithFocused(true),
		table.WithHeight(height),
	)
	return t, true
}

package tui

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// maxCSVRows limits how many rows are rendered in tabular preview.
const maxCSVRows = 100

// formatJSON pretty-prints JSON content. Returns the formatted string and true
// if the input was valid JSON, or the original content and false otherwise.
func formatJSON(content string) (string, bool) {
	content = strings.TrimSpace(content)
	if len(content) == 0 {
		return content, false
	}

	// Quick check: must start with { or [
	first := content[0]
	if first != '{' && first != '[' {
		return content, false
	}

	var raw json.RawMessage
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return content, false
	}

	formatted, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return content, false
	}

	return string(formatted), true
}

// formatCSV parses CSV content and returns a markdown-style table.
// Uses the given delimiter (comma for CSV, tab for TSV).
func formatCSV(content string, delimiter rune) (string, bool) {
	r := csv.NewReader(strings.NewReader(content))
	r.Comma = delimiter
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	// Allow variable field counts — ragged CSVs are common
	r.FieldsPerRecord = -1

	var rows [][]string
	for len(rows) <= maxCSVRows {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// If we can't parse even the first row, bail
			if len(rows) == 0 {
				return content, false
			}
			break
		}
		rows = append(rows, record)
	}

	if len(rows) == 0 {
		return content, false
	}

	return renderMarkdownTable(rows), true
}

// renderMarkdownTable converts rows into an aligned markdown table.
// The first row is treated as the header.
func renderMarkdownTable(rows [][]string) string {
	// Normalize column count to max across all rows
	maxCols := 0
	for _, row := range rows {
		if len(row) > maxCols {
			maxCols = len(row)
		}
	}
	if maxCols == 0 {
		return ""
	}

	// Calculate column widths (capped to keep things readable)
	const maxColWidth = 40
	widths := make([]int, maxCols)
	for _, row := range rows {
		for i, cell := range row {
			w := utf8.RuneCountInString(cell)
			if w > widths[i] {
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

	var buf bytes.Buffer

	// Render a single row, padding/truncating cells to column widths
	writeRow := func(row []string) {
		buf.WriteString("| ")
		for i := 0; i < maxCols; i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			runeLen := utf8.RuneCountInString(cell)
			if runeLen > widths[i] {
				// Truncate to width-1 and add ellipsis
				runes := []rune(cell)
				cell = string(runes[:widths[i]-1]) + "…"
				runeLen = widths[i]
			}
			buf.WriteString(cell)
			// Pad with spaces
			for j := runeLen; j < widths[i]; j++ {
				buf.WriteByte(' ')
			}
			buf.WriteString(" | ")
		}
		buf.WriteByte('\n')
	}

	// Header
	writeRow(rows[0])

	// Separator
	buf.WriteString("| ")
	for i := 0; i < maxCols; i++ {
		for j := 0; j < widths[i]; j++ {
			buf.WriteByte('-')
		}
		buf.WriteString(" | ")
	}
	buf.WriteByte('\n')

	// Data rows
	for _, row := range rows[1:] {
		writeRow(row)
	}

	totalRows := len(rows) - 1 // exclude header
	buf.WriteString(fmt.Sprintf("\n*%d rows displayed*", totalRows))

	return buf.String()
}

// extractYAMLFrontmatter splits YAML frontmatter from a markdown document.
// Returns the frontmatter (without delimiters), the remaining body, and
// whether frontmatter was found.
func extractYAMLFrontmatter(content string) (frontmatter, body string, ok bool) {
	// Frontmatter must start at the very beginning of the file
	trimmed := strings.TrimLeft(content, "\n\r")
	if !strings.HasPrefix(trimmed, "---") {
		return "", content, false
	}

	// Find the closing delimiter
	rest := trimmed[3:]
	// Skip the newline after opening ---
	if idx := strings.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[idx+1:]
	} else {
		return "", content, false
	}

	closeIdx := strings.Index(rest, "\n---")
	if closeIdx < 0 {
		return "", content, false
	}

	fm := strings.TrimSpace(rest[:closeIdx])
	if fm == "" {
		return "", content, false
	}

	// Body starts after the closing --- and its newline
	remaining := rest[closeIdx+4:]
	if len(remaining) > 0 && remaining[0] == '\n' {
		remaining = remaining[1:]
	}

	return fm, remaining, true
}

// renderFrontmatterCard formats extracted frontmatter as a styled block
// suitable for display above the rendered markdown.
func renderFrontmatterCard(frontmatter string) string {
	var buf bytes.Buffer
	buf.WriteString("┌─ Frontmatter ─────────────────────────┐\n")
	for _, line := range strings.Split(frontmatter, "\n") {
		buf.WriteString("│ ")
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	buf.WriteString("└───────────────────────────────────────┘\n\n")
	return buf.String()
}

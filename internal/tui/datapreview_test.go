package tui

import (
	"strings"
	"testing"
)

func TestFormatJSON_ValidObject(t *testing.T) {
	input := `{"name":"Alice","age":30,"active":true}`
	out, ok := formatJSON(input)
	if !ok {
		t.Fatal("expected ok=true for valid JSON object")
	}
	if !strings.Contains(out, "  \"name\": \"Alice\"") {
		t.Errorf("expected indented output, got:\n%s", out)
	}
}

func TestFormatJSON_ValidArray(t *testing.T) {
	input := `[1,2,3]`
	out, ok := formatJSON(input)
	if !ok {
		t.Fatal("expected ok=true for valid JSON array")
	}
	if !strings.Contains(out, "  1") {
		t.Errorf("expected indented array, got:\n%s", out)
	}
}

func TestFormatJSON_Invalid(t *testing.T) {
	_, ok := formatJSON("not json at all")
	if ok {
		t.Fatal("expected ok=false for invalid JSON")
	}
}

func TestFormatJSON_Empty(t *testing.T) {
	_, ok := formatJSON("")
	if ok {
		t.Fatal("expected ok=false for empty string")
	}
}

func TestFormatCSV_Basic(t *testing.T) {
	input := "Name,Age,City\nAlice,30,NYC\nBob,25,LA\n"
	out, ok := formatCSV(input, ',')
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strings.Contains(out, "| Name") {
		t.Errorf("expected table header, got:\n%s", out)
	}
	if !strings.Contains(out, "| Alice") {
		t.Errorf("expected data row, got:\n%s", out)
	}
	if !strings.Contains(out, "2 rows displayed") {
		t.Errorf("expected row count, got:\n%s", out)
	}
}

func TestFormatCSV_TSV(t *testing.T) {
	input := "Name\tAge\nAlice\t30\n"
	out, ok := formatCSV(input, '\t')
	if !ok {
		t.Fatal("expected ok=true for TSV")
	}
	if !strings.Contains(out, "| Name") {
		t.Errorf("expected table, got:\n%s", out)
	}
}

func TestFormatCSV_Empty(t *testing.T) {
	_, ok := formatCSV("", ',')
	if ok {
		t.Fatal("expected ok=false for empty input")
	}
}

func TestFormatCSV_RaggedRows(t *testing.T) {
	input := "A,B,C\n1,2\n3,4,5,6\n"
	out, ok := formatCSV(input, ',')
	if !ok {
		t.Fatal("expected ok=true for ragged CSV")
	}
	// Should handle variable column counts without error
	if !strings.Contains(out, "| A") {
		t.Errorf("expected table header, got:\n%s", out)
	}
}

func TestExtractYAMLFrontmatter_Present(t *testing.T) {
	input := "---\ntitle: Hello\ndate: 2024-01-01\n---\n# Body\nContent here.\n"
	fm, body, ok := extractYAMLFrontmatter(input)
	if !ok {
		t.Fatal("expected frontmatter to be found")
	}
	if !strings.Contains(fm, "title: Hello") {
		t.Errorf("expected frontmatter content, got: %s", fm)
	}
	if !strings.Contains(body, "# Body") {
		t.Errorf("expected body content, got: %s", body)
	}
}

func TestExtractYAMLFrontmatter_Missing(t *testing.T) {
	input := "# Just markdown\nNo frontmatter here.\n"
	_, _, ok := extractYAMLFrontmatter(input)
	if ok {
		t.Fatal("expected ok=false when no frontmatter")
	}
}

func TestExtractYAMLFrontmatter_UnclosedDelimiter(t *testing.T) {
	input := "---\ntitle: Hello\n# Body\n"
	_, _, ok := extractYAMLFrontmatter(input)
	if ok {
		t.Fatal("expected ok=false for unclosed frontmatter")
	}
}

func TestExtractYAMLFrontmatter_EmptyFrontmatter(t *testing.T) {
	input := "---\n---\n# Body\n"
	_, _, ok := extractYAMLFrontmatter(input)
	if ok {
		t.Fatal("expected ok=false for empty frontmatter block")
	}
}

func TestRenderFrontmatterCard(t *testing.T) {
	card := renderFrontmatterCard("title: Hello\ndate: 2024-01-01")
	if !strings.Contains(card, "Frontmatter") {
		t.Errorf("expected card header, got:\n%s", card)
	}
	if !strings.Contains(card, "title: Hello") {
		t.Errorf("expected frontmatter content in card, got:\n%s", card)
	}
}

func TestFormatCSV_MaxRows(t *testing.T) {
	// Build a CSV with 150 rows (header + 149 data rows)
	var b strings.Builder
	b.WriteString("id,value\n")
	for i := 1; i <= 149; i++ {
		b.WriteString("row,data\n")
	}
	out, ok := formatCSV(b.String(), ',')
	if !ok {
		t.Fatal("expected ok=true")
	}
	// Should cap at maxCSVRows (100) total rows including header
	if !strings.Contains(out, "rows displayed") {
		t.Errorf("expected row count, got:\n%s", out)
	}
}

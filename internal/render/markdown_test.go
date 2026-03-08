package render

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Getting Started", "getting-started"},
		{"API v2.0 Release!", "api-v20-release"},
		{"multiple   spaces", "multiple---spaces"},
		{"under_score", "under_score"},
		{"ALLCAPS", "allcaps"},
		{"", ""},
		{"123 Numbers", "123-numbers"},
		{"special!@#$%chars", "specialchars"},
		{"hyphen-already", "hyphen-already"},
		{"Unicode café résumé", "unicode-caf-rsum"},
		{"  leading trailing  ", "--leading-trailing--"},
		{"a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := Slugify(tt.input)
			if got != tt.want {
				t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHeadingSlugs(t *testing.T) {
	source := `# Introduction
## Getting Started
## Getting Started
### Details
## Getting Started
`
	slugs := HeadingSlugs(source)

	// First occurrence: "getting-started"
	if !slugs["getting-started"] {
		t.Error("expected slug 'getting-started'")
	}
	// Second occurrence: "getting-started-1"
	if !slugs["getting-started-1"] {
		t.Error("expected slug 'getting-started-1'")
	}
	// Third occurrence: "getting-started-2"
	if !slugs["getting-started-2"] {
		t.Error("expected slug 'getting-started-2'")
	}
	if !slugs["introduction"] {
		t.Error("expected slug 'introduction'")
	}
	if !slugs["details"] {
		t.Error("expected slug 'details'")
	}
	// Sanity: non-existent slug
	if slugs["nonexistent"] {
		t.Error("unexpected slug 'nonexistent'")
	}
}

func TestHeadingSlugs_Empty(t *testing.T) {
	slugs := HeadingSlugs("no headings here")
	if len(slugs) != 0 {
		t.Errorf("expected 0 slugs, got %d", len(slugs))
	}
}

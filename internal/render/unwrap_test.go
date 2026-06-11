package render

import "strings"

import "testing"

func TestUnwrapSoftBreaks(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			"list item continuation lines are joined",
			"- first line\n  second line\n  third line",
			"- first line second line third line",
		},
		{
			"separate list items stay separate",
			"- item one\n- item two",
			"- item one\n- item two",
		},
		{
			"paragraph soft wraps are joined",
			"alpha beta\ngamma delta",
			"alpha beta gamma delta",
		},
		{
			"blank line is a block boundary",
			"alpha\n\nbeta",
			"alpha\n\nbeta",
		},
		{
			"heading stays on its own line",
			"# Title\nbody text",
			"# Title\nbody text",
		},
		{
			"line is not folded into a heading-following paragraph start",
			"intro\n# Heading",
			"intro\n# Heading",
		},
		{
			"hard break (two trailing spaces) is preserved",
			"line one  \nline two",
			"line one  \nline two",
		},
		{
			"hard break (trailing backslash) is preserved",
			"line one\\\nline two",
			"line one\\\nline two",
		},
		{
			"blockquote lines are left intact",
			"> quote line one\n> quote line two",
			"> quote line one\n> quote line two",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := unwrapSoftBreaks(tt.in); got != tt.want {
				t.Errorf("unwrapSoftBreaks(%q)\n got: %q\nwant: %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestUnwrapSoftBreaks_FencedCodeUntouched(t *testing.T) {
	in := "```go\nfunc main() {\n    a := 1\n    b := 2\n}\n```"
	if got := unwrapSoftBreaks(in); got != in {
		t.Errorf("fenced code must pass through unchanged\n got: %q\nwant: %q", got, in)
	}
}

func TestSpaceListItems(t *testing.T) {
	// Tight rendered list: a blank line is inserted before the second item,
	// but not before the first (its predecessor is a paragraph already, which
	// we still gap — the key invariant is no doubled blanks and a gap between
	// adjacent items).
	in := "• one\n• two\n• three"
	got := spaceListItems(in)
	if !strings.Contains(got, "• one\n\n• two") || !strings.Contains(got, "• two\n\n• three") {
		t.Errorf("expected blank lines between items, got %q", got)
	}
}

func TestSpaceListItems_NoDoubleBlank(t *testing.T) {
	in := "intro\n\n• one\n• two"
	got := spaceListItems(in)
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("should not create triple newlines, got %q", got)
	}
	if !strings.Contains(got, "• one\n\n• two") {
		t.Errorf("expected gap between items, got %q", got)
	}
}

func TestSpaceListItems_Ordered(t *testing.T) {
	in := "1. one\n2. two"
	got := spaceListItems(in)
	if !strings.Contains(got, "1. one\n\n2. two") {
		t.Errorf("expected gap between ordered items, got %q", got)
	}
}

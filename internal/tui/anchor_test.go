package tui

import (
	"testing"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"API v2.0", "api-v20"},
		{"under_score", "under_score"},
		{"", ""},
		{"special!@#chars", "specialchars"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := slugify(tt.input)
			if got != tt.want {
				t.Errorf("slugify(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"SGR bold", "\x1b[1mBold\x1b[0m", "Bold"},
		{"SGR color", "\x1b[38;5;81mBlue\x1b[0m", "Blue"},
		{"nested SGR", "\x1b[1m\x1b[38;5;81mBoldBlue\x1b[0m\x1b[0m", "BoldBlue"},
		{"OSC hyperlink", "\x1b]8;id=123;https://example.com\x1b\\Link\x1b]8;;\x1b\\", "Link"},
		{"OSC title BEL", "\x1b]0;window title\x07text", "text"},
		{"empty", "", ""},
		{"no escapes", "just text", "just text"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripANSI(tt.input)
			if got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

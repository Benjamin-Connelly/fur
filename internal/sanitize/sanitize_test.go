package sanitize

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTerminal(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"plain", "README.md", "README.md"},
		{"unicode preserved", "café-notes.md", "café-notes.md"},
		{"emoji preserved", "🚀plan.md", "🚀plan.md"},
		{"tab kept", "a\tb", "a\tb"},
		{"csi color stripped", "\x1b[31mred\x1b[0m.md", "red.md"},
		{"csi cursor move stripped", "\x1b[2J\x1b[Hclear.md", "clear.md"},
		{"osc window title bel", "\x1b]0;pwned\x07file.md", "file.md"},
		{"osc window title st", "\x1b]0;pwned\x1b\\file.md", "file.md"},
		{"bare esc consumes next byte", "a\x1bz.md", "a.md"}, // ESC + single byte form
		{"bell dropped", "ding\x07.md", "ding.md"},
		{"carriage return dropped", "over\rwrite.md", "overwrite.md"},
		{"newline dropped", "a\nb.md", "ab.md"},
		{"del dropped", "a\x7fb.md", "ab.md"},
		{"c1 dropped", "ab.md", "ab.md"},
		{"invalid utf8 byte dropped", "a\x9bb.md", "ab.md"},
		{"nul dropped", "a\x00b.md", "ab.md"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Terminal(tt.in); got != tt.want {
				t.Errorf("Terminal(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestTerminalNeverEmitsControl is the chokepoint invariant: whatever the
// input, the output contains no ESC, no other C0/C1 control byte (tab
// excepted), and no DEL. This is the property the rest of fur relies on.
func TestTerminalNeverEmitsControl(t *testing.T) {
	inputs := []string{
		"\x1b[31m\x1b]0;x\x07\x1b[2J",
		"normal",
		"\x00\x01\x02\x1b\x07\x7f\x9b",
		"mixed\x1b[1mbold\tkept\nbad",
		strings.Repeat("\x1b[J", 1000) + "tail",
	}
	for _, in := range inputs {
		out := Terminal(in)
		for _, r := range out {
			if r == '\t' {
				continue
			}
			if r == 0x1b || r == 0x7f || (r < 0x20) || (r >= 0x80 && r <= 0x9f) {
				t.Errorf("Terminal(%q) emitted control rune %U in %q", in, r, out)
			}
		}
		if len(out) > len(in) {
			t.Errorf("Terminal(%q) grew the string: %q", in, out)
		}
	}
}

// FuzzTerminal asserts the chokepoint never panics on arbitrary bytes, always
// returns valid UTF-8, never grows the input, and never emits a control byte
// (tab excepted) — the load-bearing guarantee for every terminal write.
func FuzzTerminal(f *testing.F) {
	for _, s := range []string{"", "a", "\x1b[31m", "\x1b]0;t\x07", "café", "🚀", "\x00\x7f\x9b", "\xff\xfe"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out := Terminal(s)
		if len(out) > len(s) {
			t.Errorf("output longer than input: %q -> %q", s, out)
		}
		if !utf8.ValidString(out) && utf8.ValidString(s) {
			t.Errorf("valid input produced invalid UTF-8: %q -> %q", s, out)
		}
		for _, r := range out {
			if r == '\t' {
				continue
			}
			if r == 0x1b || r == 0x7f || r < 0x20 || (r >= 0x80 && r <= 0x9f) {
				t.Errorf("emitted control rune %U for input %q", r, s)
			}
		}
	})
}

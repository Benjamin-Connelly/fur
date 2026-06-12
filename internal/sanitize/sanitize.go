// Package sanitize is the single chokepoint for neutralizing
// attacker-controlled strings before they are written to a terminal.
//
// fur runs in shared-tenancy environments where a directory adversary can
// plant files and directories whose *names* contain ANSI/OSC/CSI escape
// sequences or other terminal control bytes. When such a name is rendered to
// the file tree, status bar, task list, or any other stdout surface, those
// bytes are interpreted by the terminal — letting the adversary move the
// cursor, rewrite earlier output, set the window title, or smuggle clipboard
// payloads (audit Chain J). Every user-controlled string MUST pass through
// Terminal before it reaches the screen.
package sanitize

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// Terminal returns s with terminal-dangerous content removed: full ANSI
// escape sequences (CSI, OSC, and bare ESC-introduced forms) and any
// remaining C0/C1 control characters. Tabs are preserved (harmless and
// common in legitimate text); everything else non-printing is dropped.
// Normal printable Unicode — including multi-byte runes in legitimate
// filenames — is preserved unchanged.
//
// The function never returns a string longer than its input and is safe to
// call on arbitrary bytes (including invalid UTF-8).
func Terminal(s string) string {
	if s == "" {
		return s
	}
	// Fast path: nothing to strip.
	if !strings.ContainsFunc(s, isStripCandidate) {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == 0x1b { // ESC: consume the whole escape sequence
			i = skipEscapeSequence(runes, i)
			continue
		}
		if r == '\t' {
			b.WriteRune(r)
			continue
		}
		// Drop invalid-UTF-8 replacement runes: writing them back would grow
		// the string (1 bad byte -> 3-byte U+FFFD) and they carry no meaning
		// in a legitimate filename.
		if r == utf8.RuneError {
			continue
		}
		if unicode.IsControl(r) || r == 0x7f || (r >= 0x80 && r <= 0x9f) {
			continue // drop C0/C1 controls and DEL
		}
		b.WriteRune(r)
	}
	return b.String()
}

// isStripCandidate reports whether r would be altered by Terminal.
func isStripCandidate(r rune) bool {
	if r == '\t' {
		return false
	}
	return r == 0x1b || r == 0x7f || r == utf8.RuneError || unicode.IsControl(r) || (r >= 0x80 && r <= 0x9f)
}

// skipEscapeSequence returns the index of the last rune consumed for the
// escape sequence that starts at runes[i] (which is ESC). Recognizes CSI
// (ESC [ … final-byte), OSC (ESC ] … BEL or ST), and the generic
// two-/three-byte forms (ESC followed by a single byte, optionally an
// intermediate). Unknown forms consume just the ESC.
func skipEscapeSequence(runes []rune, i int) int {
	n := len(runes)
	if i+1 >= n {
		return i // lone trailing ESC
	}
	switch runes[i+1] {
	case '[': // CSI: params/intermediates then a final byte 0x40–0x7e
		j := i + 2
		for j < n && runes[j] >= 0x20 && runes[j] <= 0x3f {
			j++ // parameter/intermediate bytes
		}
		if j < n && runes[j] >= 0x40 && runes[j] <= 0x7e {
			return j // final byte
		}
		return j - 1
	case ']': // OSC: terminated by BEL (0x07) or ST (ESC \)
		j := i + 2
		for j < n {
			if runes[j] == 0x07 {
				return j
			}
			if runes[j] == 0x1b && j+1 < n && runes[j+1] == '\\' {
				return j + 1
			}
			j++
		}
		return n - 1
	default:
		// ESC + single (and optional intermediate) byte, e.g. ESC c, ESC ( B.
		j := i + 1
		for j+1 < n && runes[j] >= 0x20 && runes[j] <= 0x2f {
			j++ // intermediate bytes
		}
		return j
	}
}

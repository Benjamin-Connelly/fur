package render

import (
	"regexp"
	"strings"
)

// Glamour does not reflow text inside list items: when a paragraph's parent is
// a list item it emits the content with the source's soft line breaks intact,
// so editor-wrapped continuation lines render as stranded short lines instead
// of filling the viewport width. unwrapSoftBreaks works around this by joining
// soft-wrapped lines within a block into a single logical line before glamour
// renders, letting glamour's word-wrap reflow them to the pane.
//
// It is conservative: fenced code blocks are passed through untouched, blank
// lines (block boundaries) are preserved, explicit hard breaks are kept, and a
// line is never folded into the previous one when either starts a block-level
// construct (heading, list marker, blockquote, rule, table row, fence).
func unwrapSoftBreaks(src string) string {
	lines := strings.Split(src, "\n")
	out := make([]string, 0, len(lines))

	inFence := false
	var fenceMarker byte // '`' or '~'

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if inFence {
			out = append(out, line)
			if isFence(trimmed, fenceMarker) {
				inFence = false
			}
			continue
		}
		if m, ok := fenceOpen(trimmed); ok {
			inFence = true
			fenceMarker = m
			out = append(out, line)
			continue
		}

		if len(out) > 0 && canJoin(out[len(out)-1], trimmed) {
			out[len(out)-1] = strings.TrimRight(out[len(out)-1], " \t") + " " + trimmed
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// canJoin reports whether the current line is a soft-wrapped continuation of
// prev and may be folded onto it.
func canJoin(prev, curTrimmed string) bool {
	prevTrimmed := strings.TrimSpace(prev)
	if prevTrimmed == "" || curTrimmed == "" {
		return false // blank lines are block boundaries
	}
	if endsWithHardBreak(prev) {
		return false // preserve explicit line breaks
	}
	if startsBlock(curTrimmed) {
		return false // current line opens a new block
	}
	if prevIsNonProse(prevTrimmed) {
		return false // don't fold prose into a heading/quote/table/rule
	}
	return true
}

var (
	listMarkerRe = regexp.MustCompile(`^([-*+]|\d{1,9}[.)])\s`)
	hrRe         = regexp.MustCompile(`^(?:[-*_]\s*){3,}$`)
	atxHeadingRe = regexp.MustCompile(`^#{1,6}(\s|$)`)
	setextRe     = regexp.MustCompile(`^(=+|-+)$`)
)

// startsBlock reports whether trimmed begins a block-level construct that must
// stay on its own line.
func startsBlock(trimmed string) bool {
	switch {
	case atxHeadingRe.MatchString(trimmed):
		return true
	case listMarkerRe.MatchString(trimmed):
		return true
	case strings.HasPrefix(trimmed, ">"):
		return true
	case strings.HasPrefix(trimmed, "|"):
		return true
	case hrRe.MatchString(trimmed):
		return true
	case setextRe.MatchString(trimmed):
		return true
	}
	_, isFenceLine := fenceOpen(trimmed)
	return isFenceLine
}

// prevIsNonProse reports whether prev is a block whose continuation should not
// absorb the following line (headings, blockquotes, table rows, rules).
func prevIsNonProse(prevTrimmed string) bool {
	return atxHeadingRe.MatchString(prevTrimmed) ||
		strings.HasPrefix(prevTrimmed, ">") ||
		strings.HasPrefix(prevTrimmed, "|") ||
		hrRe.MatchString(prevTrimmed) ||
		setextRe.MatchString(prevTrimmed)
}

// endsWithHardBreak reports whether a line ends with a markdown hard line break
// (two or more trailing spaces, or a trailing backslash).
func endsWithHardBreak(line string) bool {
	if strings.HasSuffix(line, "  ") {
		return true
	}
	return strings.HasSuffix(strings.TrimRight(line, " \t"), "\\")
}

// fenceOpen reports whether trimmed opens a fenced code block, returning the
// fence character.
func fenceOpen(trimmed string) (byte, bool) {
	switch {
	case strings.HasPrefix(trimmed, "```"):
		return '`', true
	case strings.HasPrefix(trimmed, "~~~"):
		return '~', true
	}
	return 0, false
}

var ansiStripRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var renderedOrderedRe = regexp.MustCompile(`^\d{1,9}[.)]\s`)

// spaceListItems inserts a blank line before each rendered list-item marker so
// items aren't visually squashed together. Glamour renders lists tight
// regardless of source looseness, so this runs on its output. A blank is added
// only before a marker line whose preceding output line is non-blank, which
// avoids doubling the gap glamour already leaves before a list.
func spaceListItems(rendered string) string {
	lines := strings.Split(rendered, "\n")
	out := make([]string, 0, len(lines)+8)
	for _, line := range lines {
		if isRenderedItemStart(line) && len(out) > 0 &&
			strings.TrimSpace(ansiStripRe.ReplaceAllString(out[len(out)-1], "")) != "" {
			out = append(out, "")
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

// isRenderedItemStart reports whether a rendered line begins a list item: after
// stripping ANSI and leading indent, it starts with the bullet glyph or an
// ordered-list marker.
func isRenderedItemStart(line string) bool {
	plain := strings.TrimLeft(ansiStripRe.ReplaceAllString(line, ""), " ")
	return strings.HasPrefix(plain, "• ") || renderedOrderedRe.MatchString(plain)
}

// isFence reports whether trimmed is a fence line of the given marker (a
// closing fence is a run of three or more of the marker character).
func isFence(trimmed string, marker byte) bool {
	if len(trimmed) < 3 {
		return false
	}
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != marker {
			return false
		}
	}
	return true
}

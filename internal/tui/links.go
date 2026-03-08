package tui

import (
	"fmt"
	"strings"

	"github.com/Benjamin-Connelly/lookit/internal/index"
)

// LinkFollowMsg is sent when the user follows a link.
type LinkFollowMsg struct {
	Target   string
	Fragment string // anchor fragment to scroll to
}

// LinkSelectMsg is sent when the user picks a link from the overlay.
type LinkSelectMsg struct {
	Target string
}

// HistoryEntry records a navigation event for back/forward.
type HistoryEntry struct {
	Path   string
	Scroll int
}

// LinkNavigator manages link following with history.
type LinkNavigator struct {
	graph   *index.LinkGraph
	history []HistoryEntry
	pos     int // current position in history

	// Link selection overlay
	showing   bool
	links     []index.Link
	linkCur   int
}

// NewLinkNavigator creates a link navigator backed by a link graph.
func NewLinkNavigator(graph *index.LinkGraph) *LinkNavigator {
	return &LinkNavigator{
		graph: graph,
		pos:   -1,
	}
}

// Navigate pushes a new entry onto the history stack.
func (n *LinkNavigator) Navigate(path string, scroll int) {
	// Truncate forward history
	if n.pos < len(n.history)-1 {
		n.history = n.history[:n.pos+1]
	}
	n.history = append(n.history, HistoryEntry{Path: path, Scroll: scroll})
	n.pos = len(n.history) - 1
}

// Back returns the previous history entry, or nil if at the beginning.
func (n *LinkNavigator) Back() *HistoryEntry {
	if n.pos <= 0 {
		return nil
	}
	n.pos--
	return &n.history[n.pos]
}

// Forward returns the next history entry, or nil if at the end.
func (n *LinkNavigator) Forward() *HistoryEntry {
	if n.pos >= len(n.history)-1 {
		return nil
	}
	n.pos++
	return &n.history[n.pos]
}

// Current returns the current history entry, or nil if empty.
func (n *LinkNavigator) Current() *HistoryEntry {
	if n.pos < 0 || n.pos >= len(n.history) {
		return nil
	}
	return &n.history[n.pos]
}

// LinksAt returns the forward links from the current file.
func (n *LinkNavigator) LinksAt(path string) []index.Link {
	return n.graph.ForwardLinks(path)
}

// BacklinksAt returns files linking to the given path.
func (n *LinkNavigator) BacklinksAt(path string) []index.Link {
	return n.graph.Backlinks(path)
}

// ShowLinks opens the link selection overlay for the given file.
// If there is exactly one link, it returns the target directly.
// Returns the single target or empty string if overlay is shown.
func (n *LinkNavigator) ShowLinks(path string) (target, fragment string) {
	links := n.graph.ForwardLinks(path)
	if len(links) == 0 {
		return "", ""
	}
	if len(links) == 1 {
		return links[0].Target, links[0].Fragment
	}
	n.showing = true
	n.links = links
	n.linkCur = 0
	return "", ""
}

// IsShowingLinks returns whether the link selection overlay is visible.
func (n *LinkNavigator) IsShowingLinks() bool {
	return n.showing
}

// CloseLinks dismisses the link selection overlay.
func (n *LinkNavigator) CloseLinks() {
	n.showing = false
	n.links = nil
	n.linkCur = 0
}

// LinkMoveUp moves the cursor up in the link selection overlay.
func (n *LinkNavigator) LinkMoveUp() {
	if n.linkCur > 0 {
		n.linkCur--
	}
}

// LinkMoveDown moves the cursor down in the link selection overlay.
func (n *LinkNavigator) LinkMoveDown() {
	if n.linkCur < len(n.links)-1 {
		n.linkCur++
	}
}

// LinkSelect returns the currently selected link target and fragment, then closes the overlay.
func (n *LinkNavigator) LinkSelect() (string, string) {
	if n.linkCur >= 0 && n.linkCur < len(n.links) {
		target := n.links[n.linkCur].Target
		fragment := n.links[n.linkCur].Fragment
		n.CloseLinks()
		return target, fragment
	}
	n.CloseLinks()
	return "", ""
}

// LinkOverlayView renders the link selection overlay.
func (n *LinkNavigator) LinkOverlayView() string {
	if !n.showing || len(n.links) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Follow link:\n")
	for i, link := range n.links {
		cursor := "  "
		if i == n.linkCur {
			cursor = "> "
		}
		label := link.Text
		if label == "" {
			label = link.Target
		}
		b.WriteString(fmt.Sprintf("%s%s -> %s\n", cursor, label, link.Target))
	}
	return b.String()
}

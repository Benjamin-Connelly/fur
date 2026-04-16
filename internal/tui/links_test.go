package tui

import (
	"testing"

	"github.com/Benjamin-Connelly/fur/internal/index"
)

func newTestGraph() *index.LinkGraph {
	g := index.NewLinkGraph()
	g.SetLinks("a.md", []index.Link{
		{Source: "a.md", Target: "b.md", Text: "link to b"},
		{Source: "a.md", Target: "c.md", Text: "link to c", Fragment: "section"},
	})
	g.SetLinks("b.md", []index.Link{
		{Source: "b.md", Target: "a.md", Text: "back to a"},
	})
	return g
}

func TestLinkNavigator_NewEmpty(t *testing.T) {
	g := index.NewLinkGraph()
	nav := NewLinkNavigator(g)
	if nav.Current() != nil {
		t.Error("new navigator should have nil current")
	}
	if nav.Back() != nil {
		t.Error("new navigator Back() should return nil")
	}
	if nav.Forward() != nil {
		t.Error("new navigator Forward() should return nil")
	}
}

func TestLinkNavigator_Navigate(t *testing.T) {
	nav := NewLinkNavigator(index.NewLinkGraph())

	nav.Navigate("a.md", 0)
	cur := nav.Current()
	if cur == nil || cur.Path != "a.md" {
		t.Fatalf("expected current=a.md, got %v", cur)
	}

	nav.Navigate("b.md", 10)
	cur = nav.Current()
	if cur == nil || cur.Path != "b.md" || cur.Scroll != 10 {
		t.Fatalf("expected current=b.md@10, got %v", cur)
	}
}

func TestLinkNavigator_BackForward(t *testing.T) {
	nav := NewLinkNavigator(index.NewLinkGraph())

	nav.Navigate("a.md", 0)
	nav.Navigate("b.md", 5)
	nav.Navigate("c.md", 10)

	// Back to b
	entry := nav.Back()
	if entry == nil || entry.Path != "b.md" {
		t.Fatalf("Back() should return b.md, got %v", entry)
	}

	// Back to a
	entry = nav.Back()
	if entry == nil || entry.Path != "a.md" {
		t.Fatalf("Back() should return a.md, got %v", entry)
	}

	// No more back
	if nav.Back() != nil {
		t.Error("Back() at beginning should return nil")
	}

	// Forward to b
	entry = nav.Forward()
	if entry == nil || entry.Path != "b.md" {
		t.Fatalf("Forward() should return b.md, got %v", entry)
	}

	// Forward to c
	entry = nav.Forward()
	if entry == nil || entry.Path != "c.md" {
		t.Fatalf("Forward() should return c.md, got %v", entry)
	}

	// No more forward
	if nav.Forward() != nil {
		t.Error("Forward() at end should return nil")
	}
}

func TestLinkNavigator_NavigateTruncatesForward(t *testing.T) {
	nav := NewLinkNavigator(index.NewLinkGraph())

	nav.Navigate("a.md", 0)
	nav.Navigate("b.md", 0)
	nav.Navigate("c.md", 0)

	// Go back to a
	nav.Back()
	nav.Back()

	// Navigate to d — should truncate b, c from forward history
	nav.Navigate("d.md", 0)

	if nav.Forward() != nil {
		t.Error("forward history should be truncated after new navigation")
	}

	entry := nav.Back()
	if entry == nil || entry.Path != "a.md" {
		t.Fatalf("Back() should return a.md, got %v", entry)
	}
}

func TestLinkNavigator_ShowLinks_Empty(t *testing.T) {
	g := index.NewLinkGraph()
	nav := NewLinkNavigator(g)
	target, fragment := nav.ShowLinks("nonexistent.md")
	if target != "" || fragment != "" {
		t.Error("ShowLinks with no links should return empty strings")
	}
	if nav.IsShowingLinks() {
		t.Error("overlay should not be showing for empty links")
	}
}

func TestLinkNavigator_ShowLinks_Single(t *testing.T) {
	g := index.NewLinkGraph()
	g.SetLinks("a.md", []index.Link{
		{Source: "a.md", Target: "b.md", Text: "only link", Fragment: "top"},
	})
	nav := NewLinkNavigator(g)
	target, fragment := nav.ShowLinks("a.md")
	if target != "b.md" || fragment != "top" {
		t.Errorf("single link should return directly: got %q %q", target, fragment)
	}
	if nav.IsShowingLinks() {
		t.Error("single link should not show overlay")
	}
}

func TestLinkNavigator_ShowLinks_Multiple(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	target, fragment := nav.ShowLinks("a.md")
	if target != "" || fragment != "" {
		t.Error("multiple links should not return target directly")
	}
	if !nav.IsShowingLinks() {
		t.Error("overlay should be showing for multiple links")
	}
}

func TestLinkNavigator_LinkCursorNavigation(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	nav.ShowLinks("a.md") // 2 links

	// Start at 0
	if nav.linkCur != 0 {
		t.Errorf("initial cursor should be 0, got %d", nav.linkCur)
	}

	// Move down
	nav.LinkMoveDown()
	if nav.linkCur != 1 {
		t.Errorf("cursor should be 1 after MoveDown, got %d", nav.linkCur)
	}

	// Clamp at bottom
	nav.LinkMoveDown()
	if nav.linkCur != 1 {
		t.Error("cursor should clamp at bottom")
	}

	// Move up
	nav.LinkMoveUp()
	if nav.linkCur != 0 {
		t.Errorf("cursor should be 0 after MoveUp, got %d", nav.linkCur)
	}

	// Clamp at top
	nav.LinkMoveUp()
	if nav.linkCur != 0 {
		t.Error("cursor should clamp at top")
	}
}

func TestLinkNavigator_LinkSelect(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	nav.ShowLinks("a.md")

	// Select first link
	target, fragment := nav.LinkSelect()
	if target != "b.md" {
		t.Errorf("expected target b.md, got %q", target)
	}
	if fragment != "" {
		t.Errorf("expected empty fragment, got %q", fragment)
	}
	if nav.IsShowingLinks() {
		t.Error("overlay should be closed after select")
	}
}

func TestLinkNavigator_LinkSelectSecond(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	nav.ShowLinks("a.md")
	nav.LinkMoveDown()

	target, fragment := nav.LinkSelect()
	if target != "c.md" || fragment != "section" {
		t.Errorf("expected c.md#section, got %q#%q", target, fragment)
	}
}

func TestLinkNavigator_CloseLinks(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	nav.ShowLinks("a.md")
	nav.CloseLinks()
	if nav.IsShowingLinks() {
		t.Error("overlay should be closed")
	}
}

func TestLinkNavigator_LinkOverlayView(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	nav.ShowLinks("a.md")

	view := nav.LinkOverlayView()
	if view == "" {
		t.Fatal("overlay view should not be empty")
	}
	if !containsStr(view, "Follow link:") {
		t.Error("overlay should contain header")
	}
	if !containsStr(view, "link to b") {
		t.Error("overlay should contain link text")
	}
	if !containsStr(view, ">") {
		t.Error("overlay should contain cursor marker")
	}
}

func TestLinkNavigator_LinkOverlayView_NotShowing(t *testing.T) {
	nav := NewLinkNavigator(index.NewLinkGraph())
	if nav.LinkOverlayView() != "" {
		t.Error("overlay view should be empty when not showing")
	}
}

func TestLinkNavigator_LinksAt(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	links := nav.LinksAt("a.md")
	if len(links) != 2 {
		t.Errorf("expected 2 forward links, got %d", len(links))
	}
}

func TestLinkNavigator_BacklinksAt(t *testing.T) {
	g := newTestGraph()
	nav := NewLinkNavigator(g)
	backlinks := nav.BacklinksAt("a.md")
	if len(backlinks) != 1 {
		t.Errorf("expected 1 backlink to a.md, got %d", len(backlinks))
	}
	if backlinks[0].Source != "b.md" {
		t.Errorf("expected backlink from b.md, got %q", backlinks[0].Source)
	}
}

// containsStr is a test helper (distinct from the production containsIgnoreCase).
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

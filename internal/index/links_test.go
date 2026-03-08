package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLinkGraph_SetAndGet(t *testing.T) {
	g := NewLinkGraph()

	links := []Link{
		{Source: "a.md", Target: "b.md", Text: "link to b", Line: 1},
		{Source: "a.md", Target: "c.md", Text: "link to c", Line: 2},
	}
	g.SetLinks("a.md", links)

	fwd := g.ForwardLinks("a.md")
	if len(fwd) != 2 {
		t.Errorf("expected 2 forward links, got %d", len(fwd))
	}

	back := g.Backlinks("b.md")
	if len(back) != 1 {
		t.Errorf("expected 1 backlink to b.md, got %d", len(back))
	}
	if back[0].Source != "a.md" {
		t.Errorf("expected backlink from a.md, got %s", back[0].Source)
	}
}

func TestLinkGraph_ReplaceLinks(t *testing.T) {
	g := NewLinkGraph()

	g.SetLinks("a.md", []Link{
		{Source: "a.md", Target: "b.md", Text: "old", Line: 1},
	})
	// Replace with new links
	g.SetLinks("a.md", []Link{
		{Source: "a.md", Target: "c.md", Text: "new", Line: 1},
	})

	// Old backlink should be gone
	if len(g.Backlinks("b.md")) != 0 {
		t.Error("old backlink to b.md should be removed")
	}
	if len(g.Backlinks("c.md")) != 1 {
		t.Error("new backlink to c.md should exist")
	}
}

func TestLinkGraph_BrokenLinks(t *testing.T) {
	g := NewLinkGraph()
	g.SetLinks("a.md", []Link{
		{Source: "a.md", Target: "exists.md", Broken: false},
		{Source: "a.md", Target: "missing.md", Broken: true},
	})

	broken := g.BrokenLinks()
	if len(broken) != 1 {
		t.Errorf("expected 1 broken link, got %d", len(broken))
	}
	if broken[0].Target != "missing.md" {
		t.Errorf("expected broken link to missing.md, got %s", broken[0].Target)
	}
}

func TestExtractLinks(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "target.md"), []byte("# Target"), 0o644)

	idx := New(dir)
	idx.Build()

	content := `# Test
[link to target](target.md)
[external](https://example.com)
[broken](nonexistent.md)
[anchor](target.md#heading)
`
	links := ExtractLinks("README.md", content, idx)

	if len(links) != 3 {
		t.Fatalf("expected 3 links (skipping external), got %d", len(links))
	}

	// target.md should resolve
	if links[0].Target != "target.md" || links[0].Broken {
		t.Errorf("target.md should resolve, broken=%v", links[0].Broken)
	}

	// nonexistent.md should be broken
	if !links[1].Broken {
		t.Error("nonexistent.md should be broken")
	}

	// anchor link should strip fragment and resolve
	if links[2].Target != "target.md" || links[2].Broken {
		t.Error("anchor link should resolve to target.md")
	}
}

func TestExtractWikilinks(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "my-page.md"), []byte("# Page"), 0o644)

	idx := New(dir)
	idx.Build()

	content := `Check [[my-page]] and [[My Page|display text]] and [[missing]]`
	links := ExtractLinks("README.md", content, idx)

	if len(links) != 3 {
		t.Fatalf("expected 3 wikilinks, got %d", len(links))
	}

	// my-page should resolve to my-page.md
	if links[0].Broken {
		t.Error("[[my-page]] should resolve to my-page.md")
	}

	// My Page with display text
	if links[1].Text != "display text" {
		t.Errorf("expected display text, got %q", links[1].Text)
	}

	// missing should be broken
	if !links[2].Broken {
		t.Error("[[missing]] should be broken")
	}
}

func TestExtractLinks_PureFragment(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "page.md"), []byte("# Heading\n## Sub"), 0o644)

	idx := New(dir)
	idx.Build()

	content := `See [this section](#sub) for details.`
	links := ExtractLinks("page.md", content, idx)

	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Target != "page.md" {
		t.Errorf("pure fragment link should target current file, got %q", links[0].Target)
	}
	if links[0].Fragment != "sub" {
		t.Errorf("expected fragment 'sub', got %q", links[0].Fragment)
	}
	if links[0].Broken {
		t.Error("pure fragment link should not be broken (file exists)")
	}
}

func TestValidateFragments(t *testing.T) {
	dir := t.TempDir()
	// Target file with known headings
	os.WriteFile(filepath.Join(dir, "target.md"), []byte("# Introduction\n## Getting Started\n## FAQ\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "source.md"), []byte("# Source\n"), 0o644)

	idx := New(dir)
	idx.Build()

	g := NewLinkGraph()
	g.SetLinks("source.md", []Link{
		{Source: "source.md", Target: "target.md", Fragment: "introduction", Text: "intro", Line: 1},
		{Source: "source.md", Target: "target.md", Fragment: "getting-started", Text: "start", Line: 2},
		{Source: "source.md", Target: "target.md", Fragment: "nonexistent-heading", Text: "bad", Line: 3},
		{Source: "source.md", Target: "target.md", Fragment: "", Text: "no frag", Line: 4},
		{Source: "source.md", Target: "missing.md", Fragment: "anything", Text: "broken file", Line: 5, Broken: true},
	})

	g.ValidateFragments(idx)

	fwd := g.ForwardLinks("source.md")
	if len(fwd) != 5 {
		t.Fatalf("expected 5 links, got %d", len(fwd))
	}

	// Valid fragment: "introduction"
	if fwd[0].BrokenFragment {
		t.Error("'introduction' fragment should be valid")
	}

	// Valid fragment: "getting-started"
	if fwd[1].BrokenFragment {
		t.Error("'getting-started' fragment should be valid")
	}

	// Invalid fragment: "nonexistent-heading"
	if !fwd[2].BrokenFragment {
		t.Error("'nonexistent-heading' fragment should be marked broken")
	}

	// No fragment: should not be marked
	if fwd[3].BrokenFragment {
		t.Error("link without fragment should not be marked broken")
	}

	// Broken target: should not be checked for fragments
	if fwd[4].BrokenFragment {
		t.Error("link with broken target should not be fragment-checked")
	}
}

func TestBrokenFragmentLinks(t *testing.T) {
	g := NewLinkGraph()
	g.SetLinks("a.md", []Link{
		{Source: "a.md", Target: "b.md", Fragment: "ok", BrokenFragment: false},
		{Source: "a.md", Target: "b.md", Fragment: "bad", BrokenFragment: true},
		{Source: "a.md", Target: "c.md", Fragment: "also-bad", BrokenFragment: true},
	})

	broken := g.BrokenFragmentLinks()
	if len(broken) != 2 {
		t.Errorf("expected 2 broken fragment links, got %d", len(broken))
	}
}

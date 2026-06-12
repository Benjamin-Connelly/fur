package render

import (
	"strings"
	"testing"
)

// TestSlugifyNFKCCollision is the Chain M regression guard.
//
// "café" can be encoded as NFC (U+00E9 é) or NFD (U+0065 e + U+0301 combining
// acute). Before NFKC normalization the two encodings slugified differently
// ("caf" vs "cafe"), so two headings that look identical produced different
// anchors — letting an attacker craft a document where a fragment link
// resolves to a different heading than the reader expects (anchor hijack /
// content swap). After normalization both collapse to one slug.
// References lookit-9py.3.17 / .3.4.
func TestSlugifyNFKCCollision(t *testing.T) {
	nfc := "café"  // café composed
	nfd := "café" // café decomposed
	if got, got2 := Slugify(nfc), Slugify(nfd); got != got2 {
		t.Errorf("NFC vs NFD slug mismatch: Slugify(%q)=%q, Slugify(%q)=%q — "+
			"normalization desync enables anchor hijack (Chain M)", nfc, got, nfd, got2)
	}

	// Compatibility forms also fold: ﬁ (U+FB01 ligature) -> "fi".
	if got := Slugify("ofﬁce"); got != "office" {
		t.Errorf("NFKC compatibility fold failed: got %q, want office", got)
	}
}

// TestAnchorSlugsDeterministic asserts duplicate disambiguation depends only
// on document order, not Go map iteration order: repeated calls on the same
// source must yield byte-identical results.
func TestAnchorSlugsDeterministic(t *testing.T) {
	src := "# Setup\n## Setup\n### Setup\n# Other\n## Setup\n"
	first := AnchorSlugs(src)
	want := []string{"setup", "setup-1", "setup-2", "other", "setup-3"}
	if strings.Join(first, ",") != strings.Join(want, ",") {
		t.Fatalf("AnchorSlugs = %v, want %v", first, want)
	}
	for i := 0; i < 50; i++ {
		got := AnchorSlugs(src)
		if strings.Join(got, ",") != strings.Join(first, ",") {
			t.Fatalf("AnchorSlugs not deterministic: run %d = %v, first = %v", i, got, first)
		}
	}
}

// TestAnchorSlugsUnique asserts no two headings ever share a slug — the
// property fragment resolution depends on.
func TestAnchorSlugsUnique(t *testing.T) {
	src := "# A\n# A\n# A\n## a\n"
	slugs := AnchorSlugs(src)
	seen := map[string]bool{}
	for _, s := range slugs {
		if seen[s] {
			t.Errorf("duplicate slug %q in %v — fragment links become ambiguous (Chain M)", s, slugs)
		}
		seen[s] = true
	}
}

// TestHeadingSlugsMatchesAnchorSlugs guards that the set and the ordered
// mapper stay in agreement (both are the single source of truth).
func TestHeadingSlugsMatchesAnchorSlugs(t *testing.T) {
	src := "# Intro\n## Intro\n# Details\n"
	set := HeadingSlugs(src)
	for _, s := range AnchorSlugs(src) {
		if !set[s] {
			t.Errorf("AnchorSlugs produced %q but HeadingSlugs set lacks it", s)
		}
	}
	if len(set) != 3 {
		t.Errorf("HeadingSlugs set size = %d, want 3", len(set))
	}
}

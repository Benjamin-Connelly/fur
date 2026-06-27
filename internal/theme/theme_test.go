package theme

import (
	"testing"

	"github.com/alecthomas/chroma/v2/styles"
)

// TestPaletteChromaStylesExist guards against typos in palette chroma names:
// every theme's Chroma must resolve to a real chroma style, otherwise code
// highlighting silently falls back and the theme looks broken.
func TestPaletteChromaStylesExist(t *testing.T) {
	for name, p := range registry {
		if p.Chroma == "" {
			t.Errorf("theme %q has an empty chroma style", name)
			continue
		}
		if styles.Get(p.Chroma) == styles.Fallback && p.Chroma != "swapoff" {
			t.Errorf("theme %q: chroma style %q not found (resolves to fallback)", name, p.Chroma)
		}
	}
}

// TestCycleOrderMembersRegistered ensures every name in the cycle is a real
// theme, so ctrl+t never lands on an unknown that falls back to dark.
func TestCycleOrderMembersRegistered(t *testing.T) {
	for _, name := range CycleOrder {
		if _, ok := registry[name]; !ok {
			t.Errorf("CycleOrder contains %q which is not in the registry", name)
		}
	}
}

func TestResolve(t *testing.T) {
	if got := Resolve("catppuccin-mocha").Name; got != "catppuccin-mocha" {
		t.Errorf("Resolve known theme: got %q", got)
	}
	if got := Resolve("nope-not-a-theme").Name; got != "dark" {
		t.Errorf("Resolve unknown should fall back to dark, got %q", got)
	}
	if got := Resolve("ascii"); got.Chroma != "" {
		t.Errorf("ascii palette should have no chroma, got %q", got.Chroma)
	}
	// "auto" resolves to one of the neutral defaults depending on terminal bg.
	if got := Resolve("auto").Name; got != "dark" && got != "light" {
		t.Errorf("Resolve(auto) = %q, want dark or light", got)
	}
}

func TestIsValid(t *testing.T) {
	for _, ok := range []string{"auto", "ascii", "dark", "light", "gruvbox", "tokyonight-moon"} {
		if !IsValid(ok) {
			t.Errorf("IsValid(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{"neon", "", "catppuccin"} {
		if IsValid(bad) {
			t.Errorf("IsValid(%q) = true, want false", bad)
		}
	}
}

func TestNextWraps(t *testing.T) {
	last := CycleOrder[len(CycleOrder)-1]
	if got := Next(last); got != CycleOrder[0] {
		t.Errorf("Next(%q) = %q, want wrap to %q", last, got, CycleOrder[0])
	}
	if got := Next("auto"); got != CycleOrder[0] {
		t.Errorf("Next(non-member) = %q, want %q", got, CycleOrder[0])
	}
}

// TestGlamourStyleInlineCode verifies inline code is colored but unadorned:
// glamour's stock styles wrap inline Code in a padded background block, which
// reads badly in a TUI. Ours uses a distinct color (pink) with no background
// and no surrounding-space padding.
func TestGlamourStyleInlineCode(t *testing.T) {
	p := Resolve("catppuccin-mocha")
	s := GlamourStyle(p)
	if s.Code.Prefix != "" || s.Code.Suffix != "" {
		t.Errorf("inline Code should have no prefix/suffix padding, got prefix=%q suffix=%q", s.Code.Prefix, s.Code.Suffix)
	}
	if s.Code.BackgroundColor != nil {
		t.Errorf("inline Code should have no background block, got %q", *s.Code.BackgroundColor)
	}
	if s.Code.Color == nil || *s.Code.Color != p.Pink {
		t.Errorf("inline Code should be colored with the palette pink (%q), got %v", p.Pink, s.Code.Color)
	}
	// Inline code must be visually distinct from bold and italic.
	if s.Code.Color != nil && s.Strong.Color != nil && *s.Code.Color == *s.Strong.Color {
		t.Error("inline Code color must differ from bold (Strong) color")
	}
	if s.Code.Color != nil && s.Emph.Color != nil && *s.Code.Color == *s.Emph.Color {
		t.Error("inline Code color must differ from italic (Emph) color")
	}
	if s.CodeBlock.Theme == "" {
		t.Error("fenced CodeBlock should delegate to the palette chroma theme")
	}
}

func TestUIForPopulated(t *testing.T) {
	ui := UIFor("dracula")
	if ui.Accent == "" || ui.Text == "" || ui.StatusBg == "" {
		t.Errorf("UIFor returned empty core colors: %+v", ui)
	}
}

// TestLightThemePaintsBackground guards the light-theme fix: light themes must
// expose a non-empty base Bg (and a glamour Document background) so the TUI
// actually paints a light surface; otherwise their dark text is invisible on a
// dark terminal. Dark themes leave Bg empty to inherit the terminal background.
func TestLightThemePaintsBackground(t *testing.T) {
	if light := UIFor("light"); light.Bg == "" {
		t.Error("light theme UI.Bg is empty; light themes must paint a background")
	}
	if dark := UIFor("dark"); dark.Bg != "" {
		t.Errorf("dark theme UI.Bg = %q, want empty (inherit terminal bg)", dark.Bg)
	}
	if bg := GlamourStyle(Resolve("light")).Document.BackgroundColor; bg == nil || *bg == "" {
		t.Error("light theme glamour Document has no BackgroundColor")
	}
	if bg := GlamourStyle(Resolve("dark")).Document.BackgroundColor; bg != nil {
		t.Errorf("dark theme glamour Document BackgroundColor = %v, want nil", *bg)
	}

	// Regression: NAMED dark themes (which, unlike the built-in "dark", carry an
	// explicit palette Bg) must NOT paint a base background either — doing so
	// striped the preview wherever the palette Bg differed from the terminal's
	// actual background (e.g. after switching themes at runtime).
	for _, name := range []string{"nord", "dracula", "gruvbox", "catppuccin-mocha", "rose-pine"} {
		if ui := UIFor(name); ui.Bg != "" {
			t.Errorf("dark theme %q UI.Bg = %q, want empty (no per-block paint)", name, ui.Bg)
		}
		if bg := GlamourStyle(Resolve(name)).Document.BackgroundColor; bg != nil {
			t.Errorf("dark theme %q glamour Document BackgroundColor = %q, want nil", name, *bg)
		}
	}
	// Named light themes must still paint, like the built-in "light".
	for _, name := range []string{"catppuccin-latte", "gruvbox-light", "solarized-light"} {
		if ui := UIFor(name); ui.Bg == "" {
			t.Errorf("light theme %q UI.Bg is empty; light themes must paint a background", name)
		}
	}
}

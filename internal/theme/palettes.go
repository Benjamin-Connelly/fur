package theme

import "github.com/charmbracelet/lipgloss"

// CycleOrder is the order ctrl+t and `:theme` step through. It begins with the
// neutral dark/light defaults, then the named families.
var CycleOrder = []string{
	"dark",
	"light",
	"catppuccin-mocha",
	"catppuccin-macchiato",
	"catppuccin-frappe",
	"catppuccin-latte",
	"gruvbox",
	"gruvbox-light",
	"dracula",
	"nord",
	"solarized-dark",
	"solarized-light",
	"rose-pine",
	"rose-pine-moon",
	"rose-pine-dawn",
	"tokyonight-night",
	"tokyonight-storm",
	"tokyonight-moon",
	"tokyonight-day",
}

// registry maps every concrete theme name to its palette. "auto" and "ascii"
// are resolved specially in Resolve and are not stored here.
var registry = map[string]Palette{
	// Neutral defaults using ANSI 256 indices, preserving fur's original look.
	"dark": {
		Name: "dark", Dark: true,
		Bg: "", Surface: "236", Overlay: "238", Text: "252", Subtle: "240",
		Red: "203", Orange: "214", Yellow: "226", Green: "114",
		Teal: "81", Blue: "75", Mauve: "62", Pink: "212",
		Chroma: "monokai",
	},
	"light": {
		Name: "light", Dark: false,
		Bg: "255", Surface: "254", Overlay: "250", Text: "236", Subtle: "243",
		Red: "124", Orange: "166", Yellow: "136", Green: "28",
		Teal: "30", Blue: "26", Mauve: "55", Pink: "162",
		Chroma: "github",
	},

	// Catppuccin — https://github.com/catppuccin/catppuccin
	"catppuccin-mocha": {
		Name: "catppuccin-mocha", Dark: true,
		Bg: "#1e1e2e", Surface: "#313244", Overlay: "#45475a", Text: "#cdd6f4", Subtle: "#a6adc8",
		Red: "#f38ba8", Orange: "#fab387", Yellow: "#f9e2af", Green: "#a6e3a1",
		Teal: "#94e2d5", Blue: "#89b4fa", Mauve: "#cba6f7", Pink: "#f5c2e7",
		Chroma: "catppuccin-mocha",
	},
	"catppuccin-macchiato": {
		Name: "catppuccin-macchiato", Dark: true,
		Bg: "#24273a", Surface: "#363a4f", Overlay: "#494d64", Text: "#cad3f5", Subtle: "#a5adcb",
		Red: "#ed8796", Orange: "#f5a97f", Yellow: "#eed49f", Green: "#a6da95",
		Teal: "#8bd5ca", Blue: "#8aadf4", Mauve: "#c6a0f6", Pink: "#f5bde6",
		Chroma: "catppuccin-macchiato",
	},
	"catppuccin-frappe": {
		Name: "catppuccin-frappe", Dark: true,
		Bg: "#303446", Surface: "#414559", Overlay: "#51576d", Text: "#c6d0f5", Subtle: "#a5adce",
		Red: "#e78284", Orange: "#ef9f76", Yellow: "#e5c890", Green: "#a6d189",
		Teal: "#81c8be", Blue: "#8caaee", Mauve: "#ca9ee6", Pink: "#f4b8e4",
		Chroma: "catppuccin-frappe",
	},
	"catppuccin-latte": {
		Name: "catppuccin-latte", Dark: false,
		Bg: "#eff1f5", Surface: "#ccd0da", Overlay: "#bcc0cc", Text: "#4c4f69", Subtle: "#6c6f85",
		Red: "#d20f39", Orange: "#fe640b", Yellow: "#df8e1d", Green: "#40a02b",
		Teal: "#179299", Blue: "#1e66f5", Mauve: "#8839ef", Pink: "#ea76cb",
		Chroma: "catppuccin-latte",
	},

	// Gruvbox — https://github.com/morhetz/gruvbox
	"gruvbox": {
		Name: "gruvbox", Dark: true,
		Bg: "#282828", Surface: "#3c3836", Overlay: "#504945", Text: "#ebdbb2", Subtle: "#a89984",
		Red: "#fb4934", Orange: "#fe8019", Yellow: "#fabd2f", Green: "#b8bb26",
		Teal: "#8ec07c", Blue: "#83a598", Mauve: "#d3869b", Pink: "#d3869b",
		Chroma: "gruvbox",
	},
	"gruvbox-light": {
		Name: "gruvbox-light", Dark: false,
		Bg: "#fbf1c7", Surface: "#ebdbb2", Overlay: "#d5c4a1", Text: "#3c3836", Subtle: "#7c6f64",
		Red: "#9d0006", Orange: "#af3a03", Yellow: "#b57614", Green: "#79740e",
		Teal: "#427b58", Blue: "#076678", Mauve: "#8f3f71", Pink: "#b16286",
		Chroma: "gruvbox-light",
	},

	// Dracula — https://draculatheme.com
	"dracula": {
		Name: "dracula", Dark: true,
		Bg: "#282a36", Surface: "#44475a", Overlay: "#6272a4", Text: "#f8f8f2", Subtle: "#bd93f9",
		Red: "#ff5555", Orange: "#ffb86c", Yellow: "#f1fa8c", Green: "#50fa7b",
		Teal: "#8be9fd", Blue: "#8be9fd", Mauve: "#bd93f9", Pink: "#ff79c6",
		Chroma: "dracula",
	},

	// Nord — https://www.nordtheme.com
	"nord": {
		Name: "nord", Dark: true,
		Bg: "#2e3440", Surface: "#3b4252", Overlay: "#4c566a", Text: "#eceff4", Subtle: "#81a1c1",
		Red: "#bf616a", Orange: "#d08770", Yellow: "#ebcb8b", Green: "#a3be8c",
		Teal: "#88c0d0", Blue: "#81a1c1", Mauve: "#b48ead", Pink: "#b48ead",
		Chroma: "nord",
	},

	// Solarized — https://ethanschoonover.com/solarized
	"solarized-dark": {
		Name: "solarized-dark", Dark: true,
		Bg: "#002b36", Surface: "#073642", Overlay: "#586e75", Text: "#839496", Subtle: "#657b83",
		Red: "#dc322f", Orange: "#cb4b16", Yellow: "#b58900", Green: "#859900",
		Teal: "#2aa198", Blue: "#268bd2", Mauve: "#6c71c4", Pink: "#d33682",
		Chroma: "solarized-dark",
	},
	"solarized-light": {
		Name: "solarized-light", Dark: false,
		Bg: "#fdf6e3", Surface: "#eee8d5", Overlay: "#93a1a1", Text: "#657b83", Subtle: "#93a1a1",
		Red: "#dc322f", Orange: "#cb4b16", Yellow: "#b58900", Green: "#859900",
		Teal: "#2aa198", Blue: "#268bd2", Mauve: "#6c71c4", Pink: "#d33682",
		Chroma: "solarized-light",
	},

	// Rosé Pine — https://rosepinetheme.com
	"rose-pine": {
		Name: "rose-pine", Dark: true,
		Bg: "#191724", Surface: "#1f1d2e", Overlay: "#26233a", Text: "#e0def4", Subtle: "#908caa",
		Red: "#eb6f92", Orange: "#f6c177", Yellow: "#f6c177", Green: "#31748f",
		Teal: "#9ccfd8", Blue: "#31748f", Mauve: "#c4a7e7", Pink: "#ebbcba",
		Chroma: "rose-pine",
	},
	"rose-pine-moon": {
		Name: "rose-pine-moon", Dark: true,
		Bg: "#232136", Surface: "#2a273f", Overlay: "#393552", Text: "#e0def4", Subtle: "#908caa",
		Red: "#eb6f92", Orange: "#f6c177", Yellow: "#f6c177", Green: "#3e8fb0",
		Teal: "#9ccfd8", Blue: "#3e8fb0", Mauve: "#c4a7e7", Pink: "#ea9a97",
		Chroma: "rose-pine-moon",
	},
	"rose-pine-dawn": {
		Name: "rose-pine-dawn", Dark: false,
		Bg: "#faf4ed", Surface: "#fffaf3", Overlay: "#f2e9e1", Text: "#575279", Subtle: "#797593",
		Red: "#b4637a", Orange: "#ea9d34", Yellow: "#ea9d34", Green: "#286983",
		Teal: "#56949f", Blue: "#286983", Mauve: "#907aa9", Pink: "#d7827e",
		Chroma: "rose-pine-dawn",
	},

	// TokyoNight — https://github.com/folke/tokyonight.nvim
	"tokyonight-night": {
		Name: "tokyonight-night", Dark: true,
		Bg: "#1a1b26", Surface: "#292e42", Overlay: "#3b4261", Text: "#c0caf5", Subtle: "#565f89",
		Red: "#f7768e", Orange: "#ff9e64", Yellow: "#e0af68", Green: "#9ece6a",
		Teal: "#7dcfff", Blue: "#7aa2f7", Mauve: "#bb9af7", Pink: "#bb9af7",
		Chroma: "tokyonight-night",
	},
	"tokyonight-storm": {
		Name: "tokyonight-storm", Dark: true,
		Bg: "#24283b", Surface: "#292e42", Overlay: "#3b4261", Text: "#c0caf5", Subtle: "#565f89",
		Red: "#f7768e", Orange: "#ff9e64", Yellow: "#e0af68", Green: "#9ece6a",
		Teal: "#7dcfff", Blue: "#7aa2f7", Mauve: "#bb9af7", Pink: "#bb9af7",
		Chroma: "tokyonight-storm",
	},
	"tokyonight-moon": {
		Name: "tokyonight-moon", Dark: true,
		Bg: "#222436", Surface: "#2f334d", Overlay: "#3b4261", Text: "#c8d3f5", Subtle: "#636da6",
		Red: "#ff757f", Orange: "#ff966c", Yellow: "#ffc777", Green: "#c3e88d",
		Teal: "#86e1fc", Blue: "#82aaff", Mauve: "#c099ff", Pink: "#c099ff",
		Chroma: "tokyonight-moon",
	},
	"tokyonight-day": {
		Name: "tokyonight-day", Dark: false,
		Bg: "#e1e2e7", Surface: "#c4c8da", Overlay: "#a8aecb", Text: "#3760bf", Subtle: "#848cb5",
		Red: "#f52a65", Orange: "#b15c00", Yellow: "#8c6c3e", Green: "#587539",
		Teal: "#007197", Blue: "#2e7de9", Mauve: "#9854f1", Pink: "#9854f1",
		Chroma: "tokyonight-day",
	},
}

// asciiPalette carries no color; the markdown renderer falls back to glamour's
// "notty" style and code highlighting is skipped for "ascii".
var asciiPalette = Palette{Name: "ascii", Dark: true, Chroma: ""}

// Resolve returns the palette for a theme name. "auto" detects the terminal
// background; "ascii" yields the no-color palette; unknown names fall back to
// the neutral dark theme.
func Resolve(name string) Palette {
	switch name {
	case "ascii":
		return asciiPalette
	case "auto":
		if lipgloss.HasDarkBackground() {
			return registry["dark"]
		}
		return registry["light"]
	}
	if p, ok := registry[name]; ok {
		return p
	}
	return registry["dark"]
}

// IsValid reports whether name is an accepted theme value.
func IsValid(name string) bool {
	switch name {
	case "auto", "ascii":
		return true
	}
	_, ok := registry[name]
	return ok
}

// Names returns all concrete theme names in cycle order, followed by the
// "auto" and "ascii" pseudo-themes. Used for config errors and completion.
func Names() []string {
	out := make([]string, 0, len(CycleOrder)+2)
	out = append(out, CycleOrder...)
	out = append(out, "auto", "ascii")
	return out
}

// Next returns the theme that follows name in CycleOrder, wrapping around.
// Names outside the cycle (e.g. "auto", "ascii") start the cycle from the top.
func Next(name string) string {
	for i, t := range CycleOrder {
		if t == name {
			return CycleOrder[(i+1)%len(CycleOrder)]
		}
	}
	return CycleOrder[0]
}

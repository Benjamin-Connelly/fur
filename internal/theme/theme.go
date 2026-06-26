// Package theme defines fur's named color themes. A single Palette per theme
// drives three rendering surfaces from one set of colors:
//
//   - glamour markdown styling (via GlamourStyle)
//   - chroma code highlighting (via Palette.Chroma)
//   - lipgloss TUI chrome (via UI)
//
// This keeps every surface visually consistent: pick a theme and the document
// body, fenced code, and the surrounding UI all share a palette.
package theme

import (
	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/lipgloss"
)

// Palette is the canonical color set for a named theme. Colors are strings
// accepted by both lipgloss and glamour: a hex value ("#cdd6f4") or an ANSI
// 256 index ("252"). Empty means "no color" (used by the ascii theme).
type Palette struct {
	Name string // canonical theme name, e.g. "catppuccin-mocha"
	Dark bool   // true for dark themes; drives the "auto" fallback

	// Surfaces, from furthest-back to text.
	Bg      string // base background
	Surface string // panel / inline-code background (one step off Bg)
	Overlay string // borders, separators, selection background
	Text    string // primary foreground
	Subtle  string // muted foreground: line numbers, hints, unfocused labels

	// Accents.
	Red    string
	Orange string
	Yellow string
	Green  string
	Teal   string // cyan / aqua
	Blue   string
	Mauve  string // purple / magenta — the primary UI accent
	Pink   string

	// Chroma is the chroma style name used for code highlighting, both for
	// standalone code files and fenced blocks inside markdown.
	Chroma string
}

// UI holds the resolved lipgloss colors for TUI chrome, derived from a Palette.
// Components read these fields instead of hardcoding palette values.
type UI struct {
	Accent   lipgloss.Color // focused pane label bg, list cursor bg, prompts
	OnAccent lipgloss.Color // text drawn on Accent
	Dim      lipgloss.Color // muted text, indicators, unfocused labels
	Border   lipgloss.Color // pane separators, horizontal rules
	Text     lipgloss.Color // default foreground
	// Bg is the base surface background. It is empty ("") on dark themes so
	// the TUI inherits the terminal background; on light themes it is a real
	// color that every content surface must paint, or the theme's dark text is
	// invisible on a dark terminal. lipgloss treats an empty color as "no
	// background", so applying it is a no-op for dark themes.
	Bg lipgloss.Color

	Dir      lipgloss.Color // directory entries in the file list
	Markdown lipgloss.Color // markdown file entries
	Filter   lipgloss.Color // active filter input

	StatusBg lipgloss.Color // status bar background
	StatusFg lipgloss.Color // status bar foreground

	Ok  lipgloss.Color // success / connected
	Err lipgloss.Color // error / disconnected

	Cursor   lipgloss.Color // cursor gutter, current-line marker
	SelectBg lipgloss.Color // visual selection background
	GuideBg  lipgloss.Color // reading-guide row background

	LinkBg lipgloss.Color // link highlight background
	Link   lipgloss.Color // link / wikilink foreground

	SearchBg lipgloss.Color // search match background
	SearchFg lipgloss.Color // search match foreground
}

// UIFor returns the resolved UI chrome colors for a theme name.
func UIFor(name string) UI {
	return uiFromPalette(Resolve(name))
}

func uiFromPalette(p Palette) UI {
	return UI{
		Accent:   lipgloss.Color(p.Mauve),
		OnAccent: lipgloss.Color(p.Bg),
		Dim:      lipgloss.Color(p.Subtle),
		Border:   lipgloss.Color(p.Overlay),
		Text:     lipgloss.Color(p.Text),
		Bg:       lipgloss.Color(p.Bg),
		Dir:      lipgloss.Color(p.Blue),
		Markdown: lipgloss.Color(p.Green),
		Filter:   lipgloss.Color(p.Orange),
		StatusBg: lipgloss.Color(p.Surface),
		StatusFg: lipgloss.Color(p.Text),
		Ok:       lipgloss.Color(p.Green),
		Err:      lipgloss.Color(p.Red),
		Cursor:   lipgloss.Color(p.Yellow),
		SelectBg: lipgloss.Color(p.Overlay),
		GuideBg:  lipgloss.Color(p.Surface),
		LinkBg:   lipgloss.Color(p.Surface),
		Link:     lipgloss.Color(p.Teal),
		SearchBg: lipgloss.Color(p.Yellow),
		SearchFg: lipgloss.Color(p.Bg),
	}
}

// GlamourStyle builds a glamour StyleConfig from a palette. Two deliberate
// departures from glamour's stock styles:
//
//   - inline Code is colored (pink) with no background block and no
//     surrounding-space padding, distinct from bold (orange) and italic
//     (yellow), instead of glamour's padded background highlight.
//   - fenced code blocks delegate to the palette's chroma theme, so they match
//     standalone code files.
func GlamourStyle(p Palette) ansi.StyleConfig {
	return ansi.StyleConfig{
		Document: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockPrefix:     "\n",
				BlockSuffix:     "\n",
				Color:           sp(p.Text),
				BackgroundColor: sp(p.Bg), // nil on dark themes; paints light themes
			},
			Margin: up(1),
		},
		BlockQuote: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:  sp(p.Subtle),
				Italic: bp(true),
			},
			Indent:      up(1),
			IndentToken: sp("│ "),
		},
		List: ansi.StyleList{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: sp(p.Text)},
			},
			LevelIndent: 2,
		},
		Heading: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				BlockSuffix: "\n",
				Color:       sp(p.Mauve),
				Bold:        bp(true),
			},
		},
		H1: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Prefix:          " ",
				Suffix:          " ",
				Color:           sp(p.Bg),
				BackgroundColor: sp(p.Mauve),
				Bold:            bp(true),
			},
		},
		H2:             ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "## ", Color: sp(p.Blue), Bold: bp(true)}},
		H3:             ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "### ", Color: sp(p.Teal), Bold: bp(true)}},
		H4:             ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "#### ", Color: sp(p.Green), Bold: bp(true)}},
		H5:             ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "##### ", Color: sp(p.Yellow)}},
		H6:             ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{Prefix: "###### ", Color: sp(p.Orange)}},
		Text:           ansi.StylePrimitive{},
		Strikethrough:  ansi.StylePrimitive{CrossedOut: bp(true)},
		Emph:           ansi.StylePrimitive{Color: sp(p.Yellow), Italic: bp(true)},
		Strong:         ansi.StylePrimitive{Color: sp(p.Orange), Bold: bp(true)},
		HorizontalRule: ansi.StylePrimitive{Color: sp(p.Overlay), Format: "\n--------\n"},
		Item:           ansi.StylePrimitive{BlockPrefix: "• "},
		Enumeration:    ansi.StylePrimitive{BlockPrefix: ". "},
		Task:           ansi.StyleTask{Ticked: "[✓] ", Unticked: "[ ] "},
		Link:           ansi.StylePrimitive{Color: sp(p.Blue), Underline: bp(true)},
		LinkText:       ansi.StylePrimitive{Color: sp(p.Teal), Bold: bp(true)},
		Image:          ansi.StylePrimitive{Color: sp(p.Blue), Underline: bp(true)},
		ImageText:      ansi.StylePrimitive{Color: sp(p.Subtle), Format: "Image: {{.text}} →"},
		Code: ansi.StyleBlock{
			// Inline code: a distinct color, no background block. Pink keeps it
			// clearly separate from bold (orange) and italic (yellow).
			StylePrimitive: ansi.StylePrimitive{
				Color: sp(p.Pink),
			},
		},
		CodeBlock: ansi.StyleCodeBlock{
			StyleBlock: ansi.StyleBlock{
				StylePrimitive: ansi.StylePrimitive{Color: sp(p.Subtle)},
				Margin:         up(1),
			},
			Theme: p.Chroma,
		},
		Table:                 ansi.StyleTable{StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{}}},
		DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n🠶 "},
	}
}

// sp returns a pointer to s, or nil if s is empty so glamour leaves the color unset.
func sp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func bp(b bool) *bool { return &b }

func up(u uint) *uint { return &u }

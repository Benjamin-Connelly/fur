package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
)

// KeyMap defines all keybindings for the TUI.
type KeyMap struct {
	Quit       key.Binding
	Up         key.Binding
	Down       key.Binding
	Enter      key.Binding
	Back       key.Binding
	Tab        key.Binding
	Search     key.Binding
	Help       key.Binding
	Follow     key.Binding
	Backlinks  key.Binding
	TOC        key.Binding
	Bookmark   key.Binding
	Command    key.Binding
	CopyLink   key.Binding
	GitInfo    key.Binding
	Copy       key.Binding
	Reload     key.Binding
	HalfUp     key.Binding
	HalfDown   key.Binding
}

// DefaultKeyMap returns the default keybinding set.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Quit:      key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		Up:        key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("k", "up")),
		Down:      key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("j", "down")),
		Enter:     key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		Back:      key.NewBinding(key.WithKeys("backspace", "h"), key.WithHelp("h", "back")),
		Tab:       key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch panel")),
		Search:    key.NewBinding(key.WithKeys("/", "ctrl+k"), key.WithHelp("/", "search")),
		Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
		Follow:    key.NewBinding(key.WithKeys("f", "gf"), key.WithHelp("f", "follow link")),
		Backlinks: key.NewBinding(key.WithKeys("b"), key.WithHelp("b", "backlinks")),
		TOC:       key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "table of contents")),
		Bookmark:  key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "bookmark")),
		Command:   key.NewBinding(key.WithKeys(":"), key.WithHelp(":", "command")),
		CopyLink:  key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy link")),
		GitInfo:   key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "git info")),
		Copy:      key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "copy file")),
		Reload:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "reload")),
		HalfUp:    key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "half-page up")),
		HalfDown:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "half-page down")),
	}
}

// VimKeyMap returns vim-style keybindings.
func VimKeyMap() KeyMap {
	km := DefaultKeyMap()
	// Vim defaults are already the default, add extras
	return km
}

// EmacsKeyMap returns emacs-style keybindings.
func EmacsKeyMap() KeyMap {
	km := DefaultKeyMap()
	km.Up = key.NewBinding(key.WithKeys("ctrl+p", "up"), key.WithHelp("C-p", "up"))
	km.Down = key.NewBinding(key.WithKeys("ctrl+n", "down"), key.WithHelp("C-n", "down"))
	km.Search = key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("C-s", "search"))
	km.Back = key.NewBinding(key.WithKeys("ctrl+b"), key.WithHelp("C-b", "back"))
	return km
}

// Help returns a formatted help string showing all keybindings.
func Help(km KeyMap) string {
	var b strings.Builder
	b.WriteString("Lookit - Key Bindings\n")
	b.WriteString(strings.Repeat("=", 40) + "\n\n")

	bindings := []key.Binding{
		km.Quit, km.Up, km.Down, km.Enter, km.Back,
		km.Tab, km.Search, km.Help, km.Follow, km.Backlinks,
		km.TOC, km.Bookmark, km.Command, km.CopyLink, km.GitInfo,
		km.Copy, km.Reload, km.HalfUp, km.HalfDown,
	}

	for _, binding := range bindings {
		h := binding.Help()
		b.WriteString(fmt.Sprintf("  %-12s %s\n", h.Key, h.Desc))
	}

	b.WriteString("\nNavigation\n")
	b.WriteString(strings.Repeat("-", 40) + "\n")
	b.WriteString("  g/G          go to top/bottom\n")
	b.WriteString("  pgup/pgdn    page up/down (preview)\n")
	b.WriteString("  ctrl+u/d     half-page up/down\n")

	b.WriteString("\nFilter Mode\n")
	b.WriteString(strings.Repeat("-", 40) + "\n")
	b.WriteString("  type         fuzzy filter files\n")
	b.WriteString("  enter        open selected\n")
	b.WriteString("  esc          clear filter\n")

	return b.String()
}

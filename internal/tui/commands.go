package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/lookit/internal/index"
)

// CommandEntry represents an entry in the command palette.
type CommandEntry struct {
	Name        string
	Description string
	Action      func() tea.Msg // returns a message to dispatch
}

// CommandPalette manages the colon-mode command interface.
type CommandPalette struct {
	commands []CommandEntry
	input    string
	filtered []CommandEntry
	cursor   int
	active   bool
}

// NewCommandPalette creates a command palette with default commands.
func NewCommandPalette() CommandPalette {
	return CommandPalette{}
}

// RegisterCommands sets up the standard command set for the TUI.
func (p *CommandPalette) RegisterCommands(idx *index.Index, links *index.LinkGraph) {
	p.RegisterCommand(CommandEntry{
		Name:        "quit",
		Description: "Exit lookit",
		Action: func() tea.Msg {
			return tea.Quit()
		},
	})

	p.RegisterCommand(CommandEntry{
		Name:        "theme dark",
		Description: "Switch to dark theme",
		Action: func() tea.Msg {
			return StatusMsg{Text: "Theme: dark (restart to apply)"}
		},
	})

	p.RegisterCommand(CommandEntry{
		Name:        "theme light",
		Description: "Switch to light theme",
		Action: func() tea.Msg {
			return StatusMsg{Text: "Theme: light (restart to apply)"}
		},
	})

	p.RegisterCommand(CommandEntry{
		Name:        "keymap vim",
		Description: "Switch to vim keybindings",
		Action: func() tea.Msg {
			return StatusMsg{Text: "Keymap: vim (restart to apply)"}
		},
	})

	p.RegisterCommand(CommandEntry{
		Name:        "keymap emacs",
		Description: "Switch to emacs keybindings",
		Action: func() tea.Msg {
			return StatusMsg{Text: "Keymap: emacs (restart to apply)"}
		},
	})

	p.RegisterCommand(CommandEntry{
		Name:        "keymap default",
		Description: "Switch to default keybindings",
		Action: func() tea.Msg {
			return StatusMsg{Text: "Keymap: default (restart to apply)"}
		},
	})

	p.RegisterCommand(CommandEntry{
		Name:        "links",
		Description: "Show all links in current file",
		Action: func() tea.Msg {
			return commandLinksMsg{}
		},
	})

	p.RegisterCommand(CommandEntry{
		Name:        "broken",
		Description: "Show all broken links",
		Action: func() tea.Msg {
			broken := links.BrokenLinks()
			if len(broken) == 0 {
				return StatusMsg{Text: "No broken links found"}
			}
			var b strings.Builder
			b.WriteString("Broken Links\n")
			b.WriteString(strings.Repeat("=", 40) + "\n\n")
			for _, link := range broken {
				b.WriteString("  " + link.Source + " -> " + link.Target)
				if link.Text != "" {
					b.WriteString(" [" + link.Text + "]")
				}
				b.WriteString("\n")
			}
			return PreviewLoadedMsg{Path: "Broken Links", Content: b.String()}
		},
	})
}

// RegisterCommand adds a command to the palette.
func (p *CommandPalette) RegisterCommand(cmd CommandEntry) {
	p.commands = append(p.commands, cmd)
}

// Open activates the command palette.
func (p *CommandPalette) Open() {
	p.active = true
	p.input = ""
	p.filtered = p.commands
	p.cursor = 0
}

// Close deactivates the command palette.
func (p *CommandPalette) Close() {
	p.active = false
	p.input = ""
}

// IsActive returns whether the palette is open.
func (p *CommandPalette) IsActive() bool {
	return p.active
}

// SetInput updates the filter input.
func (p *CommandPalette) SetInput(s string) {
	p.input = s
	p.filtered = nil
	for _, cmd := range p.commands {
		if len(p.input) == 0 || containsIgnoreCase(cmd.Name, p.input) {
			p.filtered = append(p.filtered, cmd)
		}
	}
	p.cursor = 0
}

// MoveUp moves the cursor up in the palette.
func (p *CommandPalette) MoveUp() {
	if p.cursor > 0 {
		p.cursor--
	}
}

// MoveDown moves the cursor down in the palette.
func (p *CommandPalette) MoveDown() {
	if p.cursor < len(p.filtered)-1 {
		p.cursor++
	}
}

// Execute runs the selected command and returns its message.
func (p *CommandPalette) Execute() tea.Msg {
	if p.cursor >= 0 && p.cursor < len(p.filtered) {
		if p.filtered[p.cursor].Action != nil {
			msg := p.filtered[p.cursor].Action()
			p.Close()
			return msg
		}
	}
	p.Close()
	return nil
}

// HandleOpenInput handles the "open" command with fuzzy file matching.
func (p *CommandPalette) HandleOpenInput(idx *index.Index) tea.Msg {
	query := strings.TrimSpace(strings.TrimPrefix(p.input, "open "))
	if query == "" {
		return StatusMsg{Text: "Usage: open <filename>"}
	}
	results := idx.FuzzySearch(query, 1)
	if len(results) == 0 {
		return StatusMsg{Text: "No file found matching: " + query}
	}
	p.Close()
	return FileSelectedMsg{Entry: results[0]}
}

// View renders the command palette overlay.
func (p CommandPalette) View() string {
	if !p.active {
		return ""
	}
	s := ":" + p.input + "\n"
	for i, cmd := range p.filtered {
		cursor := "  "
		if i == p.cursor {
			cursor = "> "
		}
		s += cursor + cmd.Name + " - " + cmd.Description + "\n"
	}
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	s += hintStyle.Render("↑/↓:navigate  ctrl+u:clear  ctrl+w:del word  :N jump to line  enter:run  esc:close")
	return s
}

// commandLinksMsg is an internal message to show links for the current file.
type commandLinksMsg struct{}

func containsIgnoreCase(s, substr string) bool {
	sl := len(s)
	subl := len(substr)
	if subl > sl {
		return false
	}
	for i := 0; i <= sl-subl; i++ {
		match := true
		for j := 0; j < subl; j++ {
			a, b := s[i+j], substr[j]
			if a >= 'A' && a <= 'Z' {
				a += 32
			}
			if b >= 'A' && b <= 'Z' {
				b += 32
			}
			if a != b {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

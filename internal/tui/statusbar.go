package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/Benjamin-Connelly/fur/internal/sanitize"
	"github.com/Benjamin-Connelly/fur/internal/theme"
)

// StatusBarModel renders the bottom status bar.
type StatusBarModel struct {
	filePath         string
	message          string
	mode             string
	focus            Panel
	showingHelp      bool
	linkActive       bool
	linkText         string
	visualMode       bool
	visualRange      string
	searchMode       bool
	searchQuery      string
	searchView       string // rendered search textinput (text + cursor)
	searchMatchCount int
	searchRegexErr   bool // true if regex failed to compile
	filterActive     bool // true if file list has frozen filter
	filterQuery      string
	wordCount        int
	readingTime      int // minutes
	width            int

	// Remote connection state
	remoteDisplay string // e.g. "user@host:/path"
	remoteState   string // "Connected", "Reconnecting", etc.
	lastSync      string // e.g. "5s ago"

	// ui holds the active theme's chrome colors (set by Model.applyThemeChrome).
	ui theme.UI
}

// NewStatusBarModel creates a status bar.
func NewStatusBarModel() StatusBarModel {
	return StatusBarModel{
		mode: "NORMAL",
	}
}

// SetFile updates the displayed file path. The path is sanitized of terminal
// control sequences because a directory adversary controls file and directory
// names (audit Chain J); filePath is display-only.
func (m *StatusBarModel) SetFile(path string) {
	m.filePath = sanitize.Terminal(path)
}

// SetMessage sets a temporary status message.
func (m *StatusBarModel) SetMessage(msg string) {
	m.message = msg
}

// SetMode sets the current input mode display.
func (m *StatusBarModel) SetMode(mode string) {
	m.mode = mode
}

// contextHints returns panel-specific keybinding hints.
func (m StatusBarModel) contextHints() string {
	if m.searchMode {
		// searchView renders the textinput (query text + cursor); fall back to
		// the plain query if it is unset.
		field := m.searchView
		if field == "" {
			field = m.searchQuery
		}
		if m.searchRegexErr {
			return fmt.Sprintf("/ %s  invalid regex  ctrl-r:substring  esc:close", field)
		}
		return fmt.Sprintf("/ %s (%d matches)  enter/esc:close  ctrl-r:regex  up/down:history", field, m.searchMatchCount)
	}
	if m.visualMode {
		hint := "j/k:select lines  y:copy permalink  g/G:top/bottom  esc:cancel"
		if m.visualRange != "" {
			hint = m.visualRange + "  " + hint
		}
		return hint
	}
	if m.showingHelp {
		return "esc:close help  ?:close help  q:quit"
	}
	if m.linkActive {
		hint := "tab:next link  shift-tab:prev  enter:follow  esc:clear"
		if m.linkText != "" {
			hint = m.linkText + "  " + hint
		}
		return hint
	}
	switch m.focus {
	case PanelPreview:
		return "j/k:move  /:search  n/N:match  H:guide  V:sel  t:toc  b:links  ?:help"
	case PanelSide:
		return "j/k:scroll  enter:select  d:delete  esc:back"
	default: // PanelFileList
		if m.filterActive {
			return fmt.Sprintf("[%s]  j/k:nav  enter:open  esc:clear  ?:help", m.filterQuery)
		}
		return "j/k:nav  enter:open  /:filter  t:toc  b:links  i:git  ?:help  q:quit"
	}
}

// View renders the status bar.
func (m StatusBarModel) View() string {
	modeStyle := lipgloss.NewStyle().
		Background(m.ui.Accent).
		Foreground(m.ui.OnAccent).
		Bold(true).
		Padding(0, 1)

	barStyle := lipgloss.NewStyle().
		Background(m.ui.StatusBg).
		Foreground(m.ui.StatusFg)

	hintStyle := lipgloss.NewStyle().
		Background(m.ui.StatusBg).
		Foreground(m.ui.Dim)

	modeStr := modeStyle.Render(m.mode)

	var middle string
	if m.remoteDisplay != "" {
		connStyle := lipgloss.NewStyle().
			Background(m.ui.StatusBg).
			Foreground(m.ui.Ok)
		if m.remoteState == "Reconnecting" || m.remoteState == "Disconnected" {
			connStyle = connStyle.Foreground(m.ui.Err)
		}
		middle = " " + m.remoteDisplay
		if m.remoteState != "" {
			middle += "  " + connStyle.Render(m.remoteState)
		}
		if m.lastSync != "" {
			middle += "  " + m.lastSync
		}
		if m.filePath != "" {
			middle += "  " + m.filePath
		}
	} else if m.filePath != "" {
		middle = " " + m.filePath
		if m.wordCount > 0 {
			middle += fmt.Sprintf("  %dw ~%dm", m.wordCount, m.readingTime)
		}
	}

	right := ""
	if m.message != "" {
		right = m.message + " "
	} else {
		right = hintStyle.Render(m.contextHints() + " ")
	}

	// Pad middle to fill available width
	modeWidth := lipgloss.Width(modeStr)
	rightWidth := lipgloss.Width(right)
	middleWidth := m.width - modeWidth - rightWidth
	if middleWidth < 0 {
		middleWidth = 0
	}

	paddedMiddle := barStyle.Render(fmt.Sprintf("%-*s", middleWidth, middle))

	return strings.Join([]string{modeStr, paddedMiddle, right}, "")
}

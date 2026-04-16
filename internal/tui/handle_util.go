package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) navigateToPath(path string, scroll int) (tea.Model, tea.Cmd) {
	entry := m.idx.Lookup(path)
	if entry == nil {
		return m, func() tea.Msg {
			return StatusMsg{Text: "File not found: " + path}
		}
	}

	// Update file list cursor to match
	for i, node := range m.fileList.visible {
		if node.entry.RelPath == path {
			m.fileList.cursor = i
			break
		}
	}

	return m, func() tea.Msg {
		return FileSelectedMsg{Entry: *entry}
	}
}

func (m *Model) openInEditor() (tea.Model, tea.Cmd) {
	// Determine which file to edit
	var filePath string
	if m.focus == PanelFileList {
		sel := m.fileList.SelectedVisible()
		if sel != nil && !sel.IsDir {
			filePath = sel.Path
		}
	} else if m.preview.filePath != "" {
		entry := m.idx.Lookup(m.preview.filePath)
		if entry != nil {
			filePath = entry.Path
		}
	}
	if filePath == "" {
		return m, func() tea.Msg {
			return StatusMsg{Text: "No file selected"}
		}
	}

	// Image files: open with system viewer instead of editor
	ext := filepath.Ext(filePath)
	if IsImageFile(ext) {
		return m.openWithSystem(filePath)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	c := exec.Command(editor, filePath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return StatusMsg{Text: "Editor error: " + err.Error()}
		}
		// Reload the file after editing
		entry := m.idx.Lookup(m.preview.filePath)
		if entry != nil {
			return FileSelectedMsg{Entry: *entry}
		}
		return StatusMsg{Text: "File edited"}
	})
}

func (m *Model) clearStatusAfter() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

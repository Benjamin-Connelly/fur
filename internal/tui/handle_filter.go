package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Benjamin-Connelly/fur/internal/index"
)

func (m *Model) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		// Toggle between filename and content search modes
		if m.searchMode == "filename" {
			m.searchMode = "content"
		} else {
			m.searchMode = "filename"
		}
		m.fileList.searchMode = m.searchMode
		m.applyFilter(m.fileList.filter)
		return m, nil
	case "esc":
		m.mode = modeNormal
		m.fileList.ClearFilter()
		m.searchMode = "filename"
		m.fileList.searchMode = "filename"
		m.status.SetMode("NORMAL")
		return m, nil
	case "enter":
		m.mode = modeNormal
		m.fileList.filtering = false
		m.focus = PanelFileList
		m.status.SetMode("FILES")
		return m, nil
	case "backspace":
		if len(m.fileList.filter) > 0 {
			m.applyFilter(m.fileList.filter[:len(m.fileList.filter)-1])
		}
		return m, nil
	case "up", "ctrl+p", "ctrl+k":
		m.fileList.MoveUp()
		return m, nil
	case "down", "ctrl+n", "ctrl+j":
		m.fileList.MoveDown()
		return m, nil
	case "ctrl+u":
		m.applyFilter("")
		return m, nil
	case "ctrl+w":
		// Delete last word
		input := m.fileList.filter
		input = strings.TrimRight(input, " ")
		if i := strings.LastIndex(input, " "); i >= 0 {
			m.applyFilter(input[:i+1])
		} else {
			m.applyFilter("")
		}
		return m, nil
	default:
		ch := msg.String()
		// Ignore the `/` that triggered filter mode
		if len(ch) == 1 && ch != "/" {
			m.applyFilter(m.fileList.filter + ch)
		}
		return m, nil
	}
}

func (m *Model) applyFilter(query string) {
	if m.searchMode == "content" && m.idx.GetFulltext() != nil && query != "" {
		results, err := m.idx.GetFulltext().Search(query, 50)
		if err == nil {
			entries := make([]index.FileEntry, 0, len(results))
			for _, r := range results {
				if e := m.idx.Lookup(r.Path); e != nil {
					entries = append(entries, *e)
				}
			}
			m.fileList.filter = query
			m.fileList.filtered = entries
			m.fileList.cursor = 0
			m.fileList.offset = 0
			return
		}
	}
	m.fileList.SetFilter(query)
}

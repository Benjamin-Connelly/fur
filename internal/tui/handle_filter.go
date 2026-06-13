package tui

import (
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
	case "up", "ctrl+p", "ctrl+k":
		m.fileList.MoveUp()
		return m, nil
	case "down", "ctrl+n", "ctrl+j":
		m.fileList.MoveDown()
		return m, nil
	default:
		// Everything else — printable runes, backspace, left/right, home/end,
		// ctrl+w (word delete), ctrl+u (delete to start), ctrl+a/ctrl+e — is
		// editing. Route it through the textinput, then refilter on the new
		// value. This gives proper in-place cursor editing the old append-only
		// string lacked.
		ti, cmd := m.fileList.filterInput.Update(msg)
		m.fileList.filterInput = ti
		m.applyFilter(ti.Value())
		return m, cmd
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

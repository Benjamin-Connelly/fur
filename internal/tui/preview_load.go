package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/afero"

	"github.com/Benjamin-Connelly/lookit/internal/index"
	"github.com/Benjamin-Connelly/lookit/internal/plugin"
)

// previewWithSourceMsg carries both rendered preview and raw markdown source.
type previewWithSourceMsg struct {
	preview   PreviewLoadedMsg
	rawSource string
}

func (m *Model) loadPreview(entry index.FileEntry) (tea.Model, tea.Cmd) {
	// Capture renderers for closure (safe since they're pointers)
	mdRenderer := m.mdRenderer
	codeRenderer := m.codeRenderer
	plugins := m.plugins
	fs := m.idx.Fs()

	imgRenderer := m.imageRenderer

	return m, func() tea.Msg {
		if entry.IsDir {
			return PreviewLoadedMsg{
				Path:    entry.RelPath,
				Content: "[directory]",
			}
		}

		ext := filepath.Ext(entry.RelPath)

		// Image files: text-based info card (terminal image protocols are
		// incompatible with Bubble Tea's alt-screen rendering)
		if IsImageFile(ext) {
			return PreviewLoadedMsg{
				Path:    entry.RelPath,
				Content: imgRenderer.Render(entry.Path),
			}
		}

		data, err := afero.ReadFile(fs, entry.Path)
		if err != nil {
			return PreviewLoadedMsg{
				Path:    entry.RelPath,
				Content: "Error: " + err.Error(),
			}
		}

		// Block binary files — check for null bytes in first 8KB
		sample := data
		if len(sample) > 8192 {
			sample = sample[:8192]
		}
		for _, b := range sample {
			if b == 0 {
				return PreviewLoadedMsg{
					Path:    entry.RelPath,
					Content: fmt.Sprintf("[binary file: %s]", formatFileSize(entry.Size)),
				}
			}
		}

		content := string(data)
		ext = strings.ToLower(ext)

		if entry.IsMarkdown {
			rawSource := content
			var fmCard string
			if fm, body, ok := extractYAMLFrontmatter(content); ok {
				fmCard = renderFrontmatterCard(fm)
				content = body
			}
			if plugins != nil {
				ctx := &plugin.HookContext{Content: content, FilePath: entry.RelPath}
				plugins.Run(plugin.HookBeforeRender, ctx)
				content = ctx.Content
			}
			if mdRenderer != nil {
				rendered, renderErr := mdRenderer.Render(content)
				if renderErr == nil {
					if plugins != nil {
						ctx := &plugin.HookContext{Content: rendered, FilePath: entry.RelPath}
						plugins.Run(plugin.HookAfterRender, ctx)
						rendered = ctx.Content
					}
					return previewWithSourceMsg{
						preview: PreviewLoadedMsg{
							Path:    entry.RelPath,
							Content: fmCard + rendered,
						},
						rawSource: rawSource,
					}
				}
			}
		} else if ext == ".json" {
			if formatted, ok := formatJSON(content); ok {
				highlighted, hlErr := codeRenderer.Highlight("data.json", formatted)
				if hlErr == nil {
					content = highlighted
				} else {
					content = formatted
				}
			}
		} else if ext == ".csv" || ext == ".tsv" {
			delim := ','
			if ext == ".tsv" {
				delim = '\t'
			}
			if table, ok := formatCSV(content, delim); ok {
				if mdRenderer != nil {
					rendered, renderErr := mdRenderer.Render(table)
					if renderErr == nil {
						content = rendered
					} else {
						content = table
					}
				} else {
					content = table
				}
			}
		} else if isTextFile(ext) {
			highlighted, hlErr := codeRenderer.Highlight(filepath.Base(entry.RelPath), content)
			if hlErr == nil {
				content = highlighted
			}
		}

		return PreviewLoadedMsg{
			Path:    entry.RelPath,
			Content: content,
		}
	}
}

// openWithSystem opens a file using the platform's default application.
func (m *Model) openWithSystem(filePath string) (tea.Model, tea.Cmd) {
	opener := "xdg-open" // Linux
	if _, err := exec.LookPath("open"); err == nil {
		opener = "open" // macOS
	}

	c := exec.Command(opener, filePath)
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return StatusMsg{Text: "Open error: " + err.Error()}
		}
		return StatusMsg{Text: "Opened in system viewer"}
	})
}

func isTextFile(ext string) bool {
	ext = strings.ToLower(ext)
	textExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".py": true, ".rb": true,
		".rs": true, ".c": true, ".h": true, ".cpp": true, ".java": true,
		".sh": true, ".bash": true, ".zsh": true, ".fish": true,
		".yaml": true, ".yml": true, ".toml": true, ".json": true,
		".xml": true, ".html": true, ".css": true, ".scss": true,
		".sql": true, ".lua": true, ".vim": true, ".el": true,
		".txt": true, ".cfg": true, ".ini": true, ".conf": true,
		".mk": true, ".cmake": true, ".dockerfile": true,
		".gitignore": true, ".env": true, ".mod": true, ".sum": true,
		".csv": true, ".tsv": true,
	}
	return textExts[ext]
}

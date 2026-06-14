package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// indexBuiltMsg signals that the background index build finished.
type indexBuiltMsg struct{ err error }

// loadingModel shows an animated spinner while the index builds. It quits as
// soon as the build reports via the done channel.
type loadingModel struct {
	spinner spinner.Model
	root    string
	done    <-chan error
	err     error
	built   bool
}

func (m loadingModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, waitForBuild(m.done))
}

func waitForBuild(done <-chan error) tea.Cmd {
	return func() tea.Msg { return indexBuiltMsg{err: <-done} }
}

func (m loadingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case indexBuiltMsg:
		m.built = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m loadingModel) View() string {
	style := lipgloss.NewStyle().Bold(true)
	return fmt.Sprintf("\n  %s %s\n",
		m.spinner.View(),
		style.Render("Building index for "+m.root+"…"))
}

// ShowIndexLoading runs a spinner program until buildDone delivers the index
// build result, then returns that error. Used for large trees where the
// initial walk takes long enough to warrant feedback. If the program ends
// before the build reports (e.g. ctrl+c), it still waits for the build to
// finish so the index is consistent for the caller.
func ShowIndexLoading(root string, buildDone <-chan error) error {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	final, _ := tea.NewProgram(loadingModel{spinner: sp, root: root, done: buildDone}).Run()
	if lm, ok := final.(loadingModel); ok && lm.built {
		return lm.err
	}
	return <-buildDone
}

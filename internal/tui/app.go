// Package tui wires lazycurl's panel shell and backing stores together into
// a single Bubble Tea application.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asunaro276/lazycurl/internal/collection"
	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/tui/shell"
)

// App is lazycurl's top-level Bubble Tea model. It wires the collection and
// environment stores to Shell and forwards Bubble Tea messages to it.
// Request editing happens inline within Shell's panels rather than as a
// separate top-level mode, so App itself owns no editing state.
type App struct {
	shell *shell.Shell

	width, height int
}

// New constructs the App, loading the initial collection/request state.
func New(colStore *collection.Store, envStore *environment.Store, executor *curlexec.Executor) (*App, error) {
	sh, err := shell.New(colStore, envStore, executor)
	if err != nil {
		return nil, fmt.Errorf("initializing shell: %w", err)
	}
	return &App{shell: sh}, nil
}

func (a *App) Init() tea.Cmd { return nil }

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.shell.SetSize(msg.Width, msg.Height)
	case shell.QuitMsg:
		return a, tea.Quit
	}

	cmd := a.shell.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	return a.shell.View()
}

// Package tui wires lazycurl's panel shell, request-editing form, and
// backing stores together into a single Bubble Tea application.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asunaro276/lazycurl/internal/collection"
	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/tui/form"
	"github.com/asunaro276/lazycurl/internal/tui/shell"
)

type mode int

const (
	modeShell mode = iota
	modeEditor
)

// App is lazycurl's top-level Bubble Tea model: it owns the Shell (panel
// browser) and the request-editing form, switching between them and
// persisting form saves back to the collection store.
type App struct {
	colStore *collection.Store
	envStore *environment.Store

	shell *shell.Shell

	mode           mode
	editor         form.Editor
	editingCol     string
	editingIndex   int // -1 = new request
	statusOverride string

	width, height int
}

// New constructs the App, loading the initial collection/request state.
func New(colStore *collection.Store, envStore *environment.Store, executor *curlexec.Executor) (*App, error) {
	sh, err := shell.New(colStore, envStore, executor)
	if err != nil {
		return nil, fmt.Errorf("initializing shell: %w", err)
	}
	return &App{
		colStore: colStore,
		envStore: envStore,
		shell:    sh,
		editor:   form.New(),
		mode:     modeShell,
	}, nil
}

func (a *App) Init() tea.Cmd { return nil }

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		a.shell.SetSize(msg.Width, msg.Height)
		a.editor.SetSize(msg.Width, msg.Height-4)

	case shell.QuitMsg:
		return a, tea.Quit

	case shell.OpenEditorMsg:
		a.mode = modeEditor
		a.editor = form.FromRequest(msg.Request)
		a.editor.SetSize(a.width, a.height-4)
		a.editingCol = msg.CollectionName
		a.editingIndex = msg.Index
		a.statusOverride = ""
		return a, nil

	case tea.KeyMsg:
		if a.mode == modeEditor {
			return a.updateEditor(msg)
		}
	}

	if a.mode == modeShell {
		cmd := a.shell.Update(msg)
		return a, cmd
	}
	return a, nil
}

func (a *App) updateEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		if err := a.saveEditor(); err != nil {
			a.statusOverride = err.Error()
			return a, nil
		}
		a.mode = modeShell
		return a, nil
	case "ctrl+q":
		a.mode = modeShell
		return a, nil
	}

	var cmd tea.Cmd
	a.editor, cmd = a.editor.Update(msg)
	return a, cmd
}

func (a *App) saveEditor() error {
	req := a.editor.ToRequest()
	if a.editingCol == "" {
		a.shell.UpdateAdhocRequest(req)
		return nil
	}
	requests, err := a.colStore.LoadRequests(a.editingCol)
	if err != nil {
		return err
	}
	if a.editingIndex >= 0 && a.editingIndex < len(requests) {
		requests[a.editingIndex] = req
	} else {
		requests = append(requests, req)
	}
	if err := a.colStore.SaveRequests(a.editingCol, requests); err != nil {
		return err
	}
	return a.shell.ReloadCurrentCollection()
}

var statusBarStyle = lipgloss.NewStyle().Faint(true)

func (a *App) View() string {
	if a.mode == modeEditor {
		help := "ctrl-s: 保存して戻る   ctrl-q: 破棄して戻る   tab: 次のフィールド   1-4: タブ切替   ctrl-e: 外部エディタ(Body)"
		body := a.editor.View()
		if a.statusOverride != "" {
			body += "\n\n" + a.statusOverride
		}
		return body + "\n\n" + statusBarStyle.Render(help)
	}
	return a.shell.View()
}

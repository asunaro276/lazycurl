package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asunaro276/lazycurl/internal/collection"
	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/tui/shell"
)

// stubRunner fakes curl execution for the end-to-end flow test, avoiding a
// dependency on network access or a real curl binary.
type stubRunner struct{}

func (stubRunner) Run(ctx context.Context, argv []string) ([]byte, int, error) {
	var headerFile, outFile string
	for i, a := range argv {
		if a == "-D" {
			headerFile = argv[i+1]
		}
		if a == "-o" {
			outFile = argv[i+1]
		}
	}
	_ = os.WriteFile(headerFile, []byte("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\n"), 0o600)
	_ = os.WriteFile(outFile, []byte("pong"), 0o600)
	return []byte(`{"http_code":200,"time_total":0.05}`), 0, nil
}

// runKeys feeds messages through Update without chasing any returned
// command. Sufficient for pure input (typing, navigation) where components
// like textinput may return a self-rescheduling cursor-blink command that
// isn't relevant to drive synchronously in a test.
func runKeys(t *testing.T, a *App, msgs ...tea.Msg) *App {
	t.Helper()
	for _, msg := range msgs {
		model, _ := a.Update(msg)
		a = model.(*App)
	}
	return a
}

// runKeyChase feeds a single message through Update and, if it returns a
// command, runs it once and feeds the resulting message back in. Used only
// at the specific points where Shell emits an async message (OpenEditorMsg,
// a curl-execution result) that the real bubbletea runtime would deliver on
// the next loop iteration.
func runKeyChase(t *testing.T, a *App, msg tea.Msg) *App {
	t.Helper()
	model, cmd := a.Update(msg)
	a = model.(*App)
	if cmd == nil {
		return a
	}
	next := cmd()
	if next == nil {
		return a
	}
	model, _ = a.Update(next)
	return model.(*App)
}

func runes(s string) []tea.Msg {
	var out []tea.Msg
	for _, r := range s {
		out = append(out, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return out
}

// TestPrimaryFlow exercises the MVP's primary flow end-to-end (task 8.2):
// create a collection, create a request, switch active environment, send
// the request, and confirm it appears in history.
func TestPrimaryFlow(t *testing.T) {
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutorWithRunner(stubRunner{})

	if err := envStore.Save("api", "dev", map[string]string{"host": "http://example.invalid"}); err != nil {
		t.Fatalf("pre-seed env: %v", err)
	}

	app, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	app = runKeys(t, app, tea.WindowSizeMsg{Width: 120, Height: 40})

	// lazycurl now starts in Adhoc mode; switch to Collections mode to
	// exercise the collection-creation flow below.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})

	// 1. Create collection "api".
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	msgs := runes("api")
	msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter})
	app = runKeys(t, app, msgs...)

	if len(app.shell.Collections()) != 1 || app.shell.Collections()[0].Name != "api" {
		t.Fatalf("expected collection 'api', got %+v", app.shell.Collections())
	}
	// reloadEnvironments happens on collection load, so the pre-seeded "dev"
	// environment should now be visible.

	// 2. Move focus to Requests panel and create a new request.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyTab})
	app = runKeyChase(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	if app.mode != modeEditor {
		t.Fatalf("expected editor mode after 'n', got %v", app.mode)
	}

	// Fill in the request name, then move to URL and type it.
	nameMsgs := runes("Ping")
	app = runKeys(t, app, nameMsgs...)
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyTab}) // Name -> Method
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyTab}) // Method -> URL
	urlMsgs := runes("{{host}}/ping")
	app = runKeys(t, app, urlMsgs...)

	// Save.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyCtrlS})
	if app.mode != modeShell {
		t.Fatalf("expected shell mode after save, got %v", app.mode)
	}

	requests, err := colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(requests) != 1 || requests[0].Name != "Ping" || requests[0].Method != "GET" || requests[0].URL != "{{host}}/ping" {
		t.Fatalf("unexpected saved requests: %+v", requests)
	}

	// 3. Switch active environment to "dev".
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyEnter})

	active, err := envStore.ActiveEnvironment("api")
	if err != nil {
		t.Fatalf("ActiveEnvironment: %v", err)
	}
	if active != "dev" {
		t.Fatalf("expected active env 'dev', got %q", active)
	}

	// 4. Send the request.
	app = runKeyChase(t, app, tea.KeyMsg{Type: tea.KeyEnter})

	history := app.shell.History()
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry after send, got %d: %+v", len(history), history)
	}
	if history[0].Err != nil {
		t.Fatalf("unexpected send error: %v", history[0].Err)
	}
	if history[0].Response.StatusCode != 200 || string(history[0].Response.Body) != "pong" {
		t.Fatalf("unexpected response: %+v", history[0].Response)
	}
	if history[0].Request.URL != "http://example.invalid/ping" {
		t.Fatalf("expected expanded URL, got %q", history[0].Request.URL)
	}

	// 5. Confirm history is browsable.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyTab}) // Requests -> Response
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyTab}) // Response -> History
	view := app.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
}

// TestAdhocOpenEditorMsgUpdatesScratchRequest exercises the Adhoc mode entry
// point (task 7.5): opening the editor from Adhoc mode's Editor panel (empty
// CollectionName) and saving must update the shell's in-memory scratch
// request without touching the collection store.
func TestAdhocOpenEditorMsgUpdatesScratchRequest(t *testing.T) {
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutorWithRunner(stubRunner{})

	app, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	app = runKeys(t, app, tea.WindowSizeMsg{Width: 120, Height: 40})

	// App (and Shell) start in Adhoc mode by default.
	if app.shell.Mode() != shell.ModeAdhoc {
		t.Fatalf("expected Adhoc mode by default, got %v", app.shell.Mode())
	}

	// 'e' on the Editor panel opens the form with an empty CollectionName.
	app = runKeyChase(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if app.mode != modeEditor {
		t.Fatalf("expected editor mode after 'e', got %v", app.mode)
	}
	if app.editingCol != "" {
		t.Fatalf("expected empty editingCol for Adhoc edit, got %q", app.editingCol)
	}

	nameMsgs := runes("Scratch")
	app = runKeys(t, app, nameMsgs...)
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyTab}) // Name -> Method
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyTab}) // Method -> URL
	urlMsgs := runes("https://example.com/scratch")
	app = runKeys(t, app, urlMsgs...)

	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyCtrlS})
	if app.mode != modeShell {
		t.Fatalf("expected shell mode after save, got %v", app.mode)
	}

	req := app.shell.AdhocRequest()
	if req.Name != "Scratch" || req.URL != "https://example.com/scratch" {
		t.Fatalf("unexpected scratch request: %+v", req)
	}

	cols, err := colStore.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(cols) != 0 {
		t.Fatalf("expected no collections written to disk, got %+v", cols)
	}
}

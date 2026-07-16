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
// at the specific points where Shell emits an async message (a
// curl-execution result) that the real bubbletea runtime would deliver on
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

	// lazycurl now starts on the [0] Request panel with the scratch
	// request; jump straight to [2] Collections via the digit shortcut
	// (always available while not mid-insert).
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})

	// 1. Create collection "api" ('N', uppercase, since 'n' on the
	// Collections panel now means "new request in the current collection").
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	msgs := runes("api")
	msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter})
	app = runKeys(t, app, msgs...)

	if len(app.shell.Collections()) != 1 || app.shell.Collections()[0].Name != "api" {
		t.Fatalf("expected collection 'api', got %+v", app.shell.Collections())
	}
	// The pre-seeded "dev" environment becomes visible once the collection
	// preview (which also reloads environments) is loaded.

	// 2. 'n' creates a new request in the current collection and loads it
	// directly into [0] Request, moving focus there.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	// Move to URL and type it: the form starts at Method (normal state);
	// 'j' moves to URL, enter starts insert.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Method -> URL
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyEnter})                     // start insert
	urlMsgs := runes("{{host}}/ping")
	app = runKeys(t, app, urlMsgs...)

	// Save: ctrl+s on a still-unnamed request opens the name prompt;
	// entering a name and confirming completes the save (writing the
	// in-memory requests, already updated on every keystroke, to the
	// collection's .http file).
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyCtrlS})
	nameMsgs := runes("Ping")
	app = runKeys(t, app, nameMsgs...)
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyEnter})

	requests, err := colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(requests) != 1 || requests[0].Name != "Ping" || requests[0].Method != "GET" || requests[0].URL != "{{host}}/ping" {
		t.Fatalf("unexpected saved requests: %+v", requests)
	}

	// The name-prompt overlay closed, but the URL field itself is still in
	// insert state (never explicitly exited) -- esc before digit-key panel
	// navigation, or "2" would be typed literally into the URL.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyEsc})

	// 3. Switch active environment to "dev" from the Collections panel.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyEnter})

	active, err := envStore.ActiveEnvironment("api")
	if err != nil {
		t.Fatalf("ActiveEnvironment: %v", err)
	}
	if active != "dev" {
		t.Fatalf("expected active env 'dev', got %q", active)
	}

	// 4. Back to the Request panel (still holding the saved "Ping"
	// request) and send it.
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	app = runKeyChase(t, app, tea.KeyMsg{Type: tea.KeyCtrlR})

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
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")}) // -> History
	view := app.View()
	if view == "" {
		t.Fatal("expected non-empty view")
	}
}

// TestScratchInlineEditUpdatesScratchRequest exercises the collection-less
// entry point: the [0] Request panel starts on the in-memory scratch
// request, and typing (after entering insert on URL) immediately updates
// it without touching the collection store.
func TestScratchInlineEditUpdatesScratchRequest(t *testing.T) {
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutorWithRunner(stubRunner{})

	app, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	app = runKeys(t, app, tea.WindowSizeMsg{Width: 120, Height: 40})

	// App (and Shell) start on the [0] Request panel with the scratch
	// request (Method -- Name is not part of the form; it's only
	// requested at save time).
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Method -> URL (normal)
	app = runKeys(t, app, tea.KeyMsg{Type: tea.KeyEnter})                     // start insert
	urlMsgs := runes("https://example.com/scratch")
	app = runKeys(t, app, urlMsgs...)

	req := app.shell.ScratchRequest()
	if req.URL != "https://example.com/scratch" {
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

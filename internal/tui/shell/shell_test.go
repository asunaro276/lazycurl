package shell

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asunaro276/lazycurl/internal/collection"
	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/httpfile"
)

type stubRunner struct {
	statusCode int
	body       string
}

func (r *stubRunner) Run(ctx context.Context, argv []string) ([]byte, int, error) {
	var headerFile, outFile string
	for i, a := range argv {
		if a == "-D" {
			headerFile = argv[i+1]
		}
		if a == "-o" {
			outFile = argv[i+1]
		}
	}
	_ = os.WriteFile(headerFile, []byte("HTTP/1.1 200 OK\r\n\r\n"), 0o600)
	_ = os.WriteFile(outFile, []byte(r.body), 0o600)
	return []byte(`{"http_code":200,"time_total":0.1}`), 0, nil
}

func newTestShell(t *testing.T) *Shell {
	t.Helper()
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutorWithRunner(&stubRunner{statusCode: 200, body: "ok"})

	if err := colStore.CreateCollection("api"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := colStore.CreateRequest("api", httpfile.Request{Name: "Get health", Method: "GET", URL: "https://example.com/health"}); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if err := colStore.CreateRequest("api", httpfile.Request{Name: "Get user", Method: "GET", URL: "{{host}}/users/{{id}}"}); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}

	s, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.SetSize(120, 40)
	return s
}

func TestShellLoadsCollectionsAndRequests(t *testing.T) {
	s := newTestShell(t)
	if len(s.collections) != 1 || s.collections[0].Name != "api" {
		t.Fatalf("unexpected collections: %+v", s.collections)
	}
	if len(s.requests) != 2 {
		t.Fatalf("unexpected requests: %+v", s.requests)
	}
}

func TestShellSendRequestWithoutVariables(t *testing.T) {
	s := newTestShell(t)
	s.focus = PanelRequests
	s.requestIdx = 0 // "Get health" has no {{vars}}

	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected a command from sendCurrent")
	}
	if !s.sending {
		t.Fatal("expected sending=true")
	}

	msg := cmd()
	result, ok := msg.(sendResultMsg)
	if !ok {
		t.Fatalf("expected sendResultMsg, got %T", msg)
	}
	s.Update(result)

	if s.sending {
		t.Error("expected sending=false after result")
	}
	if len(s.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(s.history))
	}
	if s.history[0].Err != nil {
		t.Errorf("unexpected error: %v", s.history[0].Err)
	}
	if s.history[0].Response.StatusCode != 200 {
		t.Errorf("unexpected status: %d", s.history[0].Response.StatusCode)
	}
}

func TestShellSendRequestWithUndefinedVariableBlocksSend(t *testing.T) {
	s := newTestShell(t)
	s.focus = PanelRequests
	s.requestIdx = 1 // "Get user" references {{host}}/{{id}}, no env active

	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected no command when variables are undefined")
	}
	if s.sending {
		t.Error("should not be sending")
	}
	if s.statusMsg == "" {
		t.Error("expected an error message about undefined variables")
	}
	if len(s.history) != 0 {
		t.Errorf("expected no history entries, got %d", len(s.history))
	}
}

func TestShellDuplicateAndDeleteRequest(t *testing.T) {
	s := newTestShell(t)
	s.focus = PanelRequests
	s.requestIdx = 0

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if len(s.requests) != 3 {
		t.Fatalf("expected 3 requests after duplicate, got %d", len(s.requests))
	}

	s.requestIdx = 2
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if s.overlay != overlayConfirmDelete {
		t.Fatalf("expected confirm-delete overlay")
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if len(s.requests) != 2 {
		t.Fatalf("expected 2 requests after delete, got %d", len(s.requests))
	}
	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after delete")
	}
}

func TestShellCreateNewCollection(t *testing.T) {
	s := newTestShell(t)
	s.focus = PanelCollections

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if s.overlay != overlayNewCollection {
		t.Fatalf("expected new-collection overlay")
	}
	for _, r := range "web" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed")
	}
	if len(s.collections) != 2 {
		t.Fatalf("expected 2 collections, got %d: %+v", len(s.collections), s.collections)
	}
}

func TestShellEnvSwitching(t *testing.T) {
	s := newTestShell(t)
	if err := s.envStore.Save("api", "dev", map[string]string{"host": "https://dev.example.com", "id": "1"}); err != nil {
		t.Fatalf("Save env: %v", err)
	}
	if err := s.reloadEnvironments(); err != nil {
		t.Fatalf("reloadEnvironments: %v", err)
	}

	s.focus = PanelRequests
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})
	if s.overlay != overlayEnvSelect {
		t.Fatalf("expected env-select overlay")
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed")
	}

	active, err := s.envStore.ActiveEnvironment("api")
	if err != nil {
		t.Fatalf("ActiveEnvironment: %v", err)
	}
	if active != "dev" {
		t.Fatalf("expected active env 'dev', got %q", active)
	}

	// Now sending the {{host}}/{{id}} request should succeed.
	s.requestIdx = 1
	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected send command once variables are defined")
	}
	msg := cmd()
	s.Update(msg)
	if len(s.history) != 1 || s.history[0].Err != nil {
		t.Fatalf("expected successful send, history=%+v", s.history)
	}
}

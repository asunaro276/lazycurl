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

func TestShellDefaultsToAdhocMode(t *testing.T) {
	s := newTestShell(t)
	if s.Mode() != ModeAdhoc {
		t.Fatalf("expected default mode ModeAdhoc, got %v", s.Mode())
	}
	if s.focus != PanelEdit {
		t.Fatalf("expected default focus PanelEdit, got %v", s.focus)
	}
}

func TestShellModeToggle(t *testing.T) {
	s := newTestShell(t)

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if s.Mode() != ModeCollections {
		t.Fatalf("expected ModeCollections after ']', got %v", s.Mode())
	}
	if s.focus != PanelCollections {
		t.Fatalf("expected focus reset to PanelCollections, got %v", s.focus)
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if s.Mode() != ModeAdhoc {
		t.Fatalf("expected ModeAdhoc after '[', got %v", s.Mode())
	}
	if s.focus != PanelEdit {
		t.Fatalf("expected focus reset to PanelEdit, got %v", s.focus)
	}
}

func TestShellAdhocEditAndSendWithoutCollection(t *testing.T) {
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutorWithRunner(&stubRunner{statusCode: 200, body: "adhoc ok"})

	s, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.SetSize(120, 40)

	if len(s.Collections()) != 0 {
		t.Fatalf("expected no collections, got %+v", s.Collections())
	}
	if s.Mode() != ModeAdhoc || s.focus != PanelEdit {
		t.Fatalf("expected Adhoc mode with PanelEdit focus at startup")
	}

	// Opening the edit form should work even with zero collections.
	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if cmd == nil {
		t.Fatal("expected OpenEditorMsg command from 'e'")
	}
	msg := cmd()
	openMsg, ok := msg.(OpenEditorMsg)
	if !ok {
		t.Fatalf("expected OpenEditorMsg, got %T", msg)
	}
	if openMsg.CollectionName != "" {
		t.Fatalf("expected empty CollectionName for Adhoc edit, got %q", openMsg.CollectionName)
	}

	// Simulate the App updating the scratch request after a form save.
	s.UpdateAdhocRequest(httpfile.Request{Name: "Adhoc ping", Method: "GET", URL: "https://example.com/ping"})

	sendCmd := s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if sendCmd == nil {
		t.Fatal("expected send command from 'enter' on PanelEdit")
	}
	if !s.sending {
		t.Fatal("expected sending=true")
	}
	result := sendCmd()
	s.Update(result)

	if s.sending {
		t.Error("expected sending=false after result")
	}
	if len(s.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(s.history))
	}
	if s.history[0].CollectionName != "" {
		t.Fatalf("expected empty CollectionName in adhoc history entry, got %q", s.history[0].CollectionName)
	}
	if s.history[0].Err != nil {
		t.Fatalf("unexpected error: %v", s.history[0].Err)
	}
}

func TestShellAdhocSaveToExistingCollection(t *testing.T) {
	s := newTestShell(t)
	s.UpdateAdhocRequest(httpfile.Request{Name: "New adhoc request", Method: "GET", URL: "https://example.com/new"})

	// 's' should be usable from any Adhoc panel, not just PanelEdit.
	s.focus = PanelResponse
	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if cmd != nil {
		t.Fatal("expected no command from 's', overlay handles the flow synchronously")
	}
	if s.overlay != overlaySaveTarget {
		t.Fatalf("expected overlaySaveTarget, got %v", s.overlay)
	}

	// index 0 = "new collection", index 1 = existing "api" collection.
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	if s.saveOverlayIdx != 1 {
		t.Fatalf("expected saveOverlayIdx 1, got %d", s.saveOverlayIdx)
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after save, got %v", s.overlay)
	}
	if s.Mode() != ModeCollections {
		t.Fatalf("expected ModeCollections after save, got %v", s.Mode())
	}
	if s.focus != PanelRequests {
		t.Fatalf("expected focus PanelRequests after save, got %v", s.focus)
	}
	if s.currentCollectionName() != "api" {
		t.Fatalf("expected 'api' collection selected, got %q", s.currentCollectionName())
	}
	if len(s.requests) != 3 || s.requests[s.requestIdx].Name != "New adhoc request" {
		t.Fatalf("expected saved request selected, requests=%+v idx=%d", s.requests, s.requestIdx)
	}

	saved, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(saved) != 3 || saved[2].Name != "New adhoc request" {
		t.Fatalf("expected request appended to disk, got %+v", saved)
	}
}

func TestShellAdhocSaveToNewCollection(t *testing.T) {
	s := newTestShell(t)
	s.UpdateAdhocRequest(httpfile.Request{Name: "Fresh request", Method: "POST", URL: "https://example.com/create"})

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	if s.overlay != overlaySaveTarget {
		t.Fatalf("expected overlaySaveTarget, got %v", s.overlay)
	}

	// index 0 ("+ new collection") is selected by default.
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if s.overlay != overlayNewCollection {
		t.Fatalf("expected overlayNewCollection, got %v", s.overlay)
	}

	for _, r := range "web" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after save, got %v", s.overlay)
	}
	if s.Mode() != ModeCollections {
		t.Fatalf("expected ModeCollections after save, got %v", s.Mode())
	}
	if s.currentCollectionName() != "web" {
		t.Fatalf("expected new 'web' collection selected, got %q", s.currentCollectionName())
	}
	if len(s.requests) != 1 || s.requests[0].Name != "Fresh request" {
		t.Fatalf("expected the adhoc request saved as the first request, got %+v", s.requests)
	}

	saved, err := s.colStore.LoadRequests("web")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(saved) != 1 || saved[0].Name != "Fresh request" {
		t.Fatalf("expected request saved to new collection file, got %+v", saved)
	}
}

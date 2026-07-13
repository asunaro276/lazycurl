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

	// A command is still returned (the status-bar auto-clear timer), but no
	// send/execution takes place.
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
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

// newEmptyTestShell returns a Shell with no collections at all, to exercise
// Adhoc mode's "no collection required" guarantees.
func newEmptyTestShell(t *testing.T) *Shell {
	t.Helper()
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutorWithRunner(&stubRunner{statusCode: 200, body: "ok"})

	s, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.SetSize(120, 40)
	return s
}

func TestShellDefaultsToAdhocMode(t *testing.T) {
	s := newTestShell(t)
	if s.Mode() != ModeAdhoc {
		t.Fatalf("expected default mode Adhoc, got %v", s.Mode())
	}
	if s.focus != PanelEditor {
		t.Fatalf("expected default focus PanelEditor, got %v", s.focus)
	}
}

func TestShellModeSwitchAndPanelCycling(t *testing.T) {
	s := newTestShell(t)
	if s.focus != PanelEditor || !s.inFormZone() {
		t.Fatalf("expected default focus PanelEditor (form zone), got focus=%v inFormZone=%v", s.focus, s.inFormZone())
	}
	if !s.editor.AtFirstFocus() {
		t.Fatal("expected the embedded form to start at its first field (Name)")
	}

	// Tabbing through the whole form (Name -> Method -> URL -> content)
	// exits forward to the next Shell panel once the form reaches its last
	// zone; a bare tab before that just moves within the form.
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // Name -> Method
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // Method -> URL
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // URL -> content
	if s.focus != PanelEditor || !s.editor.AtLastFocus() {
		t.Fatalf("expected to still be on PanelEditor at its last zone, got focus=%v", s.focus)
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // content -> exits the form
	if s.focus != PanelResponse {
		t.Fatalf("expected tab from the form's last zone to exit to PanelResponse, got %v", s.focus)
	}

	// Mode toggling via '['/']' only works outside the form zone. Response
	// is shared between both modes' panel sets, so focus doesn't move.
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	if s.Mode() != ModeCollections {
		t.Fatalf("expected Collections mode after ']', got %v", s.Mode())
	}
	if s.focus != PanelResponse {
		t.Fatalf("expected focus to remain on PanelResponse, got %v", s.focus)
	}

	// Jump to PanelCollections (not valid in Adhoc's panel set), then
	// toggle back: this should reset focus to PanelEditor and reload the
	// embedded form from the (unchanged) scratch request.
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if s.focus != PanelCollections {
		t.Fatalf("expected PanelCollections after '1', got %v", s.focus)
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	if s.Mode() != ModeAdhoc {
		t.Fatalf("expected Adhoc mode after '[', got %v", s.Mode())
	}
	if s.focus != PanelEditor {
		t.Fatalf("expected focus reset to PanelEditor, got %v", s.focus)
	}
	if !s.editor.AtFirstFocus() {
		t.Fatal("expected the embedded form to reset to Name on re-entering PanelEditor")
	}
}

func TestShellAdhocEditAndSendWithoutCollection(t *testing.T) {
	s := newEmptyTestShell(t)
	if len(s.collections) != 0 {
		t.Fatalf("expected no collections, got %+v", s.collections)
	}
	if s.Mode() != ModeAdhoc {
		t.Fatalf("expected Adhoc mode, got %v", s.Mode())
	}

	// The Editor panel is always in its form zone -- there is no separate
	// key to "enter" editing; typing directly edits the scratch request.
	if !s.inFormZone() {
		t.Fatal("expected Adhoc's Editor panel to always be in the form zone")
	}
	for _, r := range "Scratch" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // Name -> Method
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // Method -> URL
	for _, r := range "https://example.com/scratch" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if s.AdhocRequest().URL != "https://example.com/scratch" {
		t.Fatalf("expected scratch request to be updated, got %+v", s.AdhocRequest())
	}

	// ctrl+r sends the scratch request without any variable expansion
	// ('enter' is reserved for in-field editing while the form has focus).
	sendCmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	if sendCmd == nil {
		t.Fatal("expected a send command")
	}
	if !s.sending {
		t.Fatal("expected sending=true")
	}
	result := sendCmd()
	s.Update(result)

	if len(s.history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(s.history))
	}
	if s.history[0].CollectionName != "" {
		t.Fatalf("expected empty CollectionName in history entry, got %q", s.history[0].CollectionName)
	}
	if s.history[0].Err != nil {
		t.Fatalf("unexpected send error: %v", s.history[0].Err)
	}
}

func TestShellAdhocSaveToExistingCollection(t *testing.T) {
	s := newTestShell(t) // has collection "api" with 2 requests
	s.SetAdhocRequest(httpfile.Request{Name: "Scratch", Method: "GET", URL: "https://example.com/scratch"})

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.overlay != overlaySaveAdhoc {
		t.Fatalf("expected overlaySaveAdhoc, got %v", s.overlay)
	}
	s.saveIdx = 0 // "api" is the only collection

	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	requests, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(requests) != 3 || requests[2].URL != "https://example.com/scratch" {
		t.Fatalf("expected scratch request appended to 'api', got %+v", requests)
	}
	if s.AdhocRequest().URL != "" {
		t.Fatalf("expected scratch request reset after save, got %+v", s.AdhocRequest())
	}
}

func TestShellAdhocSaveToNewCollection(t *testing.T) {
	s := newTestShell(t)
	s.SetAdhocRequest(httpfile.Request{Name: "Scratch", Method: "GET", URL: "https://example.com/scratch"})

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	s.saveIdx = len(s.collections) // the "+ new collection" entry
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if s.overlay != overlayNewCollection {
		t.Fatalf("expected overlayNewCollection, got %v", s.overlay)
	}

	for _, r := range "scratchpad" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	requests, err := s.colStore.LoadRequests("scratchpad")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(requests) != 1 || requests[0].URL != "https://example.com/scratch" {
		t.Fatalf("expected scratch request saved as first request of new collection, got %+v", requests)
	}
}

// TestShellFormZoneProtectsTextInputFromGlobalShortcuts guards against a
// regression where global shortcuts (q/s/[/]/digits/enter) would swallow
// characters meant for the URL field instead of being typed literally, once
// the Requests/Editor panel's form zone has focus.
func TestShellFormZoneProtectsTextInputFromGlobalShortcuts(t *testing.T) {
	s := newTestShell(t)
	if !s.inFormZone() {
		t.Fatal("expected Adhoc's Editor panel to start in the form zone")
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // Name -> Method
	s.handleKey(tea.KeyMsg{Type: tea.KeyTab}) // Method -> URL

	for _, r := range "https://x.test/a[1]?q=1" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	want := "https://x.test/a[1]?q=1"
	if got := s.AdhocRequest().URL; got != want {
		t.Fatalf("expected literal text typed into URL, got %q want %q", got, want)
	}
	if s.sending {
		t.Fatal("'q'/'s' typed as text must not trigger quit/save side effects")
	}
	if s.overlay != overlayNone {
		t.Fatalf("expected no overlay opened by typed text, got %v", s.overlay)
	}
}

func TestShellFormZoneCtrlCStillCancelsQuit(t *testing.T) {
	s := newTestShell(t)
	if !s.inFormZone() {
		t.Fatal("expected Adhoc's Editor panel to start in the form zone")
	}
	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected ctrl+c to still produce a quit command from within the form zone")
	}
	if _, ok := cmd().(QuitMsg); !ok {
		t.Fatal("expected ctrl+c to emit QuitMsg from within the form zone")
	}
}

func TestShellSetStatusEmptyReturnsNoCommand(t *testing.T) {
	s := newTestShell(t)
	if cmd := s.setStatus(""); cmd != nil {
		t.Fatal("expected nil command when clearing status directly")
	}
}

func TestShellStatusMessageAutoClearByGeneration(t *testing.T) {
	s := newTestShell(t)

	cmd := s.setStatus("first error")
	if cmd == nil {
		t.Fatal("expected a clear-timer command for a non-empty status")
	}
	staleGen := s.statusGen

	s.setStatus("second error")
	if s.statusMsg != "second error" {
		t.Fatalf("expected latest message, got %q", s.statusMsg)
	}

	// A stale clear from the first message's timer must not affect the
	// newer message that replaced it.
	s.Update(clearStatusMsg{gen: staleGen})
	if s.statusMsg != "second error" {
		t.Fatalf("stale clear should not affect current message, got %q", s.statusMsg)
	}

	// The current message's own clear should take effect.
	s.Update(clearStatusMsg{gen: s.statusGen})
	if s.statusMsg != "" {
		t.Fatalf("expected statusMsg cleared, got %q", s.statusMsg)
	}
}

func TestShellAdhocSaveSwitchesToCollectionsMode(t *testing.T) {
	s := newTestShell(t)
	s.SetAdhocRequest(httpfile.Request{Name: "Scratch", Method: "GET", URL: "https://example.com/scratch"})

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	s.saveIdx = 0 // "api"
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if s.Mode() != ModeCollections {
		t.Fatalf("expected Collections mode after save, got %v", s.Mode())
	}
	if s.focus != PanelRequests {
		t.Fatalf("expected focus on PanelRequests after save, got %v", s.focus)
	}
	if s.currentCollectionName() != "api" {
		t.Fatalf("expected selected collection 'api', got %q", s.currentCollectionName())
	}
	if s.requestIdx != len(s.requests)-1 || s.requests[s.requestIdx].URL != "https://example.com/scratch" {
		t.Fatalf("expected selected request to be the saved one, got idx=%d requests=%+v", s.requestIdx, s.requests)
	}
	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after save, got %v", s.overlay)
	}
}

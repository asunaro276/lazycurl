package shell

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// newEmptyTestShell returns a Shell with no collections at all, to exercise
// the scratch (collection-less) request's "no collection required"
// guarantees.
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

// fakeStreamRunner delivers a fixed sequence of chunks over a channel,
// simulating a streaming curl process without spawning one. If cancelable
// is set, it blocks on ctx.Done() after delivering its chunks instead of
// returning immediately, so tests can assert that cancellation (not mere
// exhaustion of chunks) is what ends the stream.
type fakeStreamRunner struct {
	statusCode int
	chunks     [][]byte
	cancelable bool
}

func (f *fakeStreamRunner) Run(ctx context.Context, argv []string, chunks chan<- []byte) (int, error) {
	defer close(chunks)
	var headerFile string
	for i, a := range argv {
		if a == "-D" {
			headerFile = argv[i+1]
		}
	}
	_ = os.WriteFile(headerFile, []byte("HTTP/1.1 200 OK\r\n\r\n"), 0o600)
	for _, c := range f.chunks {
		select {
		case chunks <- c:
		case <-ctx.Done():
			return 0, nil
		}
	}
	if f.cancelable {
		<-ctx.Done()
	}
	return 0, nil
}

func newStreamingTestShell(t *testing.T, runner *fakeStreamRunner) *Shell {
	t.Helper()
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutorWithRunners(&stubRunner{statusCode: 200, body: "ok"}, runner)

	s, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.SetSize(120, 40)
	return s
}

// TestShellStreamSendRendersChunksIncrementallyThenConfirmsHistory exercises
// the subscribe pattern end to end: each streamChunkMsg re-arms
// listenStream, the Response panel shows the accumulated body while sending,
// and the terminal streamDoneMsg confirms a single History entry.
func TestShellStreamSendRendersChunksIncrementallyThenConfirmsHistory(t *testing.T) {
	s := newStreamingTestShell(t, &fakeStreamRunner{statusCode: 200, chunks: [][]byte{[]byte("chunk-a"), []byte("chunk-b")}})
	s.SetScratchRequest(httpfile.Request{
		Method:  "GET",
		URL:     "https://example.com/events",
		Pragmas: httpfile.Pragmas{Stream: true},
	})

	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("expected a send command")
	}
	if !s.sending {
		t.Fatal("expected sending=true")
	}
	if s.liveResponse == nil {
		t.Fatal("expected liveResponse to be initialized for a streaming send")
	}

	msg := cmd()
	chunkMsg, ok := msg.(streamChunkMsg)
	if !ok {
		t.Fatalf("expected streamChunkMsg, got %T", msg)
	}
	cmd = s.Update(chunkMsg)
	if string(s.liveResponse.Body) != "chunk-a" {
		t.Fatalf("expected first chunk appended to liveResponse, got %q", s.liveResponse.Body)
	}
	if !strings.Contains(s.viewResponse(), "chunk-a") {
		t.Fatalf("expected viewResponse to show the live chunk, got %q", s.viewResponse())
	}

	msg = cmd()
	chunkMsg, ok = msg.(streamChunkMsg)
	if !ok {
		t.Fatalf("expected second streamChunkMsg, got %T", msg)
	}
	cmd = s.Update(chunkMsg)
	if string(s.liveResponse.Body) != "chunk-achunk-b" {
		t.Fatalf("expected second chunk appended, got %q", s.liveResponse.Body)
	}

	msg = cmd()
	doneMsg, ok := msg.(streamDoneMsg)
	if !ok {
		t.Fatalf("expected streamDoneMsg, got %T", msg)
	}
	s.Update(doneMsg)

	if s.sending {
		t.Error("expected sending=false after streamDoneMsg")
	}
	if s.liveResponse != nil {
		t.Error("expected liveResponse cleared after streamDoneMsg")
	}
	if len(s.history) != 1 {
		t.Fatalf("expected exactly 1 history entry, got %d", len(s.history))
	}
	if s.history[0].Err != nil {
		t.Errorf("unexpected error: %v", s.history[0].Err)
	}
	if string(s.history[0].Response.Body) != "chunk-achunk-b" {
		t.Errorf("expected concatenated body in history, got %q", s.history[0].Response.Body)
	}
}

// TestShellStreamCancelConfirmsPartialBodyToHistory exercises ctrl-c
// cancellation mid-stream: the partial body received so far must still be
// confirmed into History, and not surfaced as an error.
func TestShellStreamCancelConfirmsPartialBodyToHistory(t *testing.T) {
	s := newStreamingTestShell(t, &fakeStreamRunner{statusCode: 200, chunks: [][]byte{[]byte("partial")}, cancelable: true})
	s.SetScratchRequest(httpfile.Request{
		Method:  "GET",
		URL:     "https://example.com/events",
		Pragmas: httpfile.Pragmas{Stream: true},
	})

	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	msg := cmd()
	chunkMsg := msg.(streamChunkMsg)
	cmd = s.Update(chunkMsg)

	// ctrl-c cancels the send's context; the fakeStreamRunner then closes
	// its chunks channel and the final Done event carries the partial body.
	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})

	msg = cmd()
	doneMsg, ok := msg.(streamDoneMsg)
	if !ok {
		t.Fatalf("expected streamDoneMsg after cancellation, got %T", msg)
	}
	s.Update(doneMsg)

	if len(s.history) != 1 {
		t.Fatalf("expected 1 history entry after cancellation, got %d", len(s.history))
	}
	if s.history[0].Err != nil {
		t.Errorf("expected cancellation to not be surfaced as an error, got %v", s.history[0].Err)
	}
	if string(s.history[0].Response.Body) != "partial" {
		t.Errorf("expected partial body confirmed to history, got %q", s.history[0].Response.Body)
	}
}

func TestShellLoadsCollectionsAndPreview(t *testing.T) {
	s := newTestShell(t)
	if len(s.collections) != 1 || s.collections[0].Name != "api" {
		t.Fatalf("unexpected collections: %+v", s.collections)
	}
	if len(s.previewRequests) != 2 {
		t.Fatalf("unexpected preview requests: %+v", s.previewRequests)
	}
}

func TestShellDefaultsToRequestPanelWithScratchRequest(t *testing.T) {
	s := newTestShell(t)
	if s.focus != PanelRequest {
		t.Fatalf("expected default focus PanelRequest, got %v", s.focus)
	}
	if !s.usingScratch {
		t.Fatal("expected a fresh Shell to start on the scratch request")
	}
}

func TestShellSendLoadedRequestWithoutVariables(t *testing.T) {
	s := newTestShell(t)
	reqs, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	s.loadRequestIntoEditor("api", reqs, 0) // "Get health" has no {{vars}}

	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("expected a command from sendLoadedCurrent")
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

func TestShellSendLoadedRequestWithUndefinedVariableBlocksSend(t *testing.T) {
	s := newTestShell(t)
	reqs, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	s.loadRequestIntoEditor("api", reqs, 1) // "Get user" references {{host}}/{{id}}, no env active

	// A command is still returned (the status-bar auto-clear timer), but no
	// send/execution takes place.
	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
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

func TestShellCollectionsDuplicateAndDeleteRequest(t *testing.T) {
	s := newTestShell(t)
	s.setFocus(PanelCollections)
	s.collectionReqIdx = 0

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	if len(s.previewRequests) != 3 {
		t.Fatalf("expected 3 requests after duplicate, got %d", len(s.previewRequests))
	}

	s.collectionReqIdx = 2
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	if s.overlay != overlayConfirmDelete {
		t.Fatalf("expected confirm-delete overlay")
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	if len(s.previewRequests) != 2 {
		t.Fatalf("expected 2 requests after delete, got %d", len(s.previewRequests))
	}
	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after delete")
	}
}

func TestShellCreateNewCollection(t *testing.T) {
	s := newTestShell(t)
	s.setFocus(PanelCollections)

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
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
	if err := s.reloadCollectionPreview(); err != nil {
		t.Fatalf("reloadCollectionPreview: %v", err)
	}

	s.setFocus(PanelCollections)
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
	reqs, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	s.loadRequestIntoEditor("api", reqs, 1)
	s.setFocus(PanelRequest)
	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
	if cmd == nil {
		t.Fatal("expected send command once variables are defined")
	}
	msg := cmd()
	s.Update(msg)
	if len(s.history) != 1 || s.history[0].Err != nil {
		t.Fatalf("expected successful send, history=%+v", s.history)
	}
}

func TestShellScratchInlineEditUpdatesScratchRequest(t *testing.T) {
	s := newEmptyTestShell(t)
	if len(s.collections) != 0 {
		t.Fatalf("expected no collections, got %+v", s.collections)
	}
	if !s.usingScratch {
		t.Fatal("expected the Request panel to start on the scratch request")
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Method -> URL (normal state)
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})                     // start insert on URL
	for _, r := range "https://example.com/scratch" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if s.ScratchRequest().URL != "https://example.com/scratch" {
		t.Fatalf("expected scratch request to be updated, got %+v", s.ScratchRequest())
	}

	// ctrl+r sends the scratch request without any variable expansion,
	// even while the URL field is still in insert state.
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

func TestShellScratchSaveToExistingCollection(t *testing.T) {
	s := newTestShell(t) // has collection "api" with 2 requests
	s.SetScratchRequest(httpfile.Request{Name: "Scratch", Method: "GET", URL: "https://example.com/scratch"})

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.overlay != overlaySaveTo {
		t.Fatalf("expected overlaySaveTo, got %v", s.overlay)
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
	if s.ScratchRequest().URL != "" {
		t.Fatalf("expected scratch request reset after save, got %+v", s.ScratchRequest())
	}
}

func TestShellScratchSaveToNewCollection(t *testing.T) {
	s := newTestShell(t)
	s.SetScratchRequest(httpfile.Request{Name: "Scratch", Method: "GET", URL: "https://example.com/scratch"})

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

func TestShellScratchSaveFocusesCollectionsOnSavedRequest(t *testing.T) {
	s := newTestShell(t)
	s.SetScratchRequest(httpfile.Request{Name: "Scratch", Method: "GET", URL: "https://example.com/scratch"})

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	s.saveIdx = 0 // "api"
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if s.focus != PanelCollections {
		t.Fatalf("expected focus on PanelCollections after save, got %v", s.focus)
	}
	if s.currentCollectionName() != "api" {
		t.Fatalf("expected selected collection 'api', got %q", s.currentCollectionName())
	}
	if s.collectionReqIdx != len(s.previewRequests)-1 || s.previewRequests[s.collectionReqIdx].URL != "https://example.com/scratch" {
		t.Fatalf("expected selected request to be the saved one, got idx=%d requests=%+v", s.collectionReqIdx, s.previewRequests)
	}
	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after save, got %v", s.overlay)
	}
	if !s.usingScratch || s.ScratchRequest().URL != "" {
		t.Fatalf("expected scratch request reset to empty after save, got %+v usingScratch=%v", s.ScratchRequest(), s.usingScratch)
	}
}

// TestShellSaveUnnamedLoadedRequestPromptsForName covers the loaded (already
// collection-bound) request save path: a freshly created request has no
// name, so ctrl+s must open overlayRequestName instead of writing to disk
// immediately; entering a name and confirming completes the save with that
// name.
func TestShellSaveUnnamedLoadedRequestPromptsForName(t *testing.T) {
	s := newTestShell(t)
	s.setFocus(PanelCollections)

	// 'n' appends a nameless request to the current collection and loads
	// it directly into [0] Request, moving focus there.
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if s.focus != PanelRequest {
		t.Fatalf("expected 'n' to move focus to PanelRequest, got %v", s.focus)
	}
	if s.requests[s.requestIdx].Name != "" {
		t.Fatalf("expected the newly created request to start unnamed, got %+v", s.requests[s.requestIdx])
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Method -> URL
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})                     // start insert
	for _, r := range "https://example.com/new" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.overlay != overlayRequestName {
		t.Fatalf("expected overlayRequestName for an unnamed request, got %v", s.overlay)
	}

	for _, r := range "New request" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after confirming the name, got %v", s.overlay)
	}

	requests, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(requests) != 3 || requests[2].Name != "New request" || requests[2].URL != "https://example.com/new" {
		t.Fatalf("expected the new request saved with the prompted name, got %+v", requests)
	}
}

// TestShellSaveUnnamedRequestPromptCancel confirms esc closes the name
// prompt without writing anything to disk.
func TestShellSaveUnnamedRequestPromptCancel(t *testing.T) {
	s := newTestShell(t)
	s.setFocus(PanelCollections)

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.overlay != overlayRequestName {
		t.Fatalf("expected overlayRequestName, got %v", s.overlay)
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	if s.overlay != overlayNone {
		t.Fatalf("expected overlay closed after esc, got %v", s.overlay)
	}

	requests, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected on-disk requests unchanged after cancel, got %+v", requests)
	}
}

// TestShellSaveNamedRequestSkipsPrompt covers the "既存リクエストの再保存"
// scenario: a request that already has a name must save immediately on
// ctrl+s, with no name prompt interrupting the flow.
func TestShellSaveNamedRequestSkipsPrompt(t *testing.T) {
	s := newTestShell(t)
	reqs, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	s.loadRequestIntoEditor("api", reqs, 0) // "Get health" already has a name

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})

	if s.overlay != overlayNone {
		t.Fatalf("expected no name prompt for an already-named request, got overlay %v", s.overlay)
	}
}

// TestShellScratchSaveUnnamedRequestPromptsForNameFirst covers the scratch
// save path: an unnamed scratch request must be named via
// overlayRequestName before the save-to-collection picker (overlaySaveTo)
// appears.
func TestShellScratchSaveUnnamedRequestPromptsForNameFirst(t *testing.T) {
	s := newTestShell(t) // has collection "api" with 2 requests
	s.SetScratchRequest(httpfile.Request{Method: "GET", URL: "https://example.com/scratch"})

	s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.overlay != overlayRequestName {
		t.Fatalf("expected overlayRequestName before the collection picker, got %v", s.overlay)
	}

	for _, r := range "Scratch" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if s.overlay != overlaySaveTo {
		t.Fatalf("expected overlaySaveTo once the name is confirmed, got %v", s.overlay)
	}
	if s.ScratchRequest().Name != "Scratch" {
		t.Fatalf("expected the scratch request to carry the prompted name, got %+v", s.ScratchRequest())
	}

	s.saveIdx = 0 // "api"
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	requests, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(requests) != 3 || requests[2].Name != "Scratch" {
		t.Fatalf("expected the scratch request saved with the prompted name, got %+v", requests)
	}
}

// --- Panel-switching / insert-state gating (0-3, tab) ---

// TestShellDigitKeysSwitchPanelsInNormalState confirms 0-3 always jump
// panels while the Request panel is in its normal (non-insert) state.
func TestShellDigitKeysSwitchPanelsInNormalState(t *testing.T) {
	s := newTestShell(t)
	if s.focus != PanelRequest || s.editor.Editing() {
		t.Fatalf("expected default focus PanelRequest in normal state, got focus=%v editing=%v", s.focus, s.editor.Editing())
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if s.focus != PanelCollections {
		t.Fatalf("expected '2' to jump to PanelCollections, got %v", s.focus)
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if s.focus != PanelHistory {
		t.Fatalf("expected '3' to jump to PanelHistory, got %v", s.focus)
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("0")})
	if s.focus != PanelRequest {
		t.Fatalf("expected '0' to jump back to PanelRequest, got %v", s.focus)
	}
}

// TestShellDigitKeysAreLiteralTextWhileInsert guards against a regression
// where global shortcuts (digits included) would swallow characters meant
// for a text field once the Request panel's editor has entered insert.
func TestShellDigitKeysAreLiteralTextWhileInsert(t *testing.T) {
	s := newTestShell(t)
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // Method -> URL
	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})                     // start insert
	if !s.editor.Editing() {
		t.Fatal("expected the editor to be in insert state")
	}

	for _, r := range "https://x.test/a123" {
		s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	want := "https://x.test/a123"
	if got := s.ScratchRequest().URL; got != want {
		t.Fatalf("expected literal digits typed into URL, got %q want %q", got, want)
	}
	if s.focus != PanelRequest {
		t.Fatalf("expected focus to stay on PanelRequest while typing digits, got %v", s.focus)
	}
}

func TestShellFormZoneCtrlCStillCancelsQuit(t *testing.T) {
	s := newTestShell(t)
	cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("expected ctrl+c to still produce a quit command from the Request panel")
	}
	if _, ok := cmd().(QuitMsg); !ok {
		t.Fatal("expected ctrl+c to emit QuitMsg from the Request panel")
	}
}

// --- Collections drilldown ---

// TestShellCollectionsDrilldownLoadsRequestIntoRequestPanel exercises the
// two-level accordion: cursor movement expands/collapses which collection
// is shown, and enter on a request row loads it into [0] Request with focus
// moving there.
func TestShellCollectionsDrilldownLoadsRequestIntoRequestPanel(t *testing.T) {
	s := newTestShell(t)
	s.setFocus(PanelCollections)
	if s.collectionReqIdx != -1 {
		t.Fatalf("expected the Collections cursor to start on the collection header, got %d", s.collectionReqIdx)
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // header -> first request row
	if s.collectionReqIdx != 0 {
		t.Fatalf("expected cursor movement to descend into the first request row, got %d", s.collectionReqIdx)
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // -> second request row
	if s.collectionReqIdx != 1 {
		t.Fatalf("expected cursor movement to the second request row, got %d", s.collectionReqIdx)
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if s.focus != PanelRequest {
		t.Fatalf("expected enter on a request row to move focus to PanelRequest, got %v", s.focus)
	}
	if s.usingScratch {
		t.Fatal("expected the loaded request to no longer be the scratch request")
	}
	if s.requests[s.requestIdx].Name != "Get user" {
		t.Fatalf("expected 'Get user' to be loaded, got %+v", s.requests[s.requestIdx])
	}
}

// TestShellCollectionsCursorMovesBetweenCollections confirms moving the
// cursor past the last request row (or up past the header) crosses into
// the neighboring collection, collapsing the previous one.
func TestShellCollectionsCursorMovesBetweenCollections(t *testing.T) {
	s := newTestShell(t)
	if err := s.colStore.CreateCollection("web"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := s.reloadCollections(); err != nil {
		t.Fatalf("reloadCollections: %v", err)
	}
	s.setFocus(PanelCollections)

	// Descend through "api"'s 2 requests, then one more down should cross
	// into "web"'s header.
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // row 0
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // row 1
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // -> next collection header
	if s.collectionIdx != 1 || s.collectionReqIdx != -1 {
		t.Fatalf("expected cursor to land on the next collection's header, got collectionIdx=%d collectionReqIdx=%d", s.collectionIdx, s.collectionReqIdx)
	}
	if s.currentCollectionName() != "web" {
		t.Fatalf("expected 'web' to be current, got %q", s.currentCollectionName())
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // back up into "api"'s last row
	if s.collectionIdx != 0 {
		t.Fatalf("expected cursor to move back to 'api', got collectionIdx=%d", s.collectionIdx)
	}
	if s.collectionReqIdx != len(s.previewRequests)-1 {
		t.Fatalf("expected cursor to land on api's last request row, got %d (previewRequests=%+v)", s.collectionReqIdx, s.previewRequests)
	}
}

// --- History preview / confirm ---

// TestShellHistoryPreviewExpandsWithoutAffectingResponseUntilEnter confirms
// j/k moves the History panel's preview cursor (viewingIdx unaffected)
// while enter confirms it into [1] Response.
func TestShellHistoryPreviewExpandsWithoutAffectingResponseUntilEnter(t *testing.T) {
	s := newTestShell(t)
	reqs, err := s.colStore.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	s.loadRequestIntoEditor("api", reqs, 0)
	for range 2 {
		cmd := s.handleKey(tea.KeyMsg{Type: tea.KeyCtrlR})
		if cmd == nil {
			t.Fatal("expected a send command")
		}
		s.Update(cmd())
	}
	if len(s.history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(s.history))
	}
	// A fresh send leaves viewingIdx at -1 (live/latest).
	if s.viewingIdx != -1 {
		t.Fatalf("expected viewingIdx to reset to -1 (live) after a send, got %d", s.viewingIdx)
	}

	s.setFocus(PanelHistory)
	s.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // preview the older entry
	if s.historyIdx != 0 {
		t.Fatalf("expected historyIdx to move to 0, got %d", s.historyIdx)
	}
	if s.viewingIdx != -1 {
		t.Fatal("expected viewingIdx to stay untouched by cursor movement alone")
	}

	s.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	if s.viewingIdx != 0 {
		t.Fatalf("expected enter to confirm viewingIdx to the previewed entry, got %d", s.viewingIdx)
	}
}

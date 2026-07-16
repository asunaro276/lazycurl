// Package shell implements lazycurl's TUI shell: the always-visible
// Request/Response/Collections/History panel grid, lazygit-compatible
// keybindings, and colored status/method badges.
package shell

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asunaro276/lazycurl/internal/collection"
	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/httpfile"
	"github.com/asunaro276/lazycurl/internal/tui/form"
)

// Panel identifies one of the shell's four always-visible panels, fixed in
// both number and position: [0] Request (top-left), [1] Response
// (top-right), [2] Collections (bottom-left), [3] History (bottom-right).
type Panel int

const (
	PanelRequest Panel = iota
	PanelResponse
	PanelCollections
	PanelHistory
)

const panelCount = 4

var panelLabels = map[Panel]string{
	PanelRequest:     "Request",
	PanelResponse:    "Response",
	PanelCollections: "Collections",
	PanelHistory:     "History",
}

// HistoryEntry records one executed request/response pair.
type HistoryEntry struct {
	CollectionName string
	Request        httpfile.Request // as sent (variables expanded)
	Response       *curlexec.Response
	Err            error
	At             time.Time
}

// overlay identifies a modal shown above the panel layout.
type overlay int

const (
	overlayNone overlay = iota
	overlayHelp
	overlayEnvSelect
	overlayNewCollection
	overlayConfirmDelete
	overlaySaveTo
	overlayRequestName
)

// QuitMsg is emitted when the user requests to quit.
type QuitMsg struct{}

// Shell is the main panel-based TUI: Request, Response, Collections, and
// History panels, always shown together and navigated with lazygit-style
// keybindings.
type Shell struct {
	colStore *collection.Store
	envStore *environment.Store
	executor *curlexec.Executor

	collections   []collection.Collection
	collectionIdx int // Collections panel: which collection header the cursor rests on; that collection is always the one shown expanded

	// collectionReqIdx is the Collections panel's cursor within the
	// expanded collection's request list: -1 means the cursor sits on the
	// collection's own header row, >=0 indexes into previewRequests.
	collectionReqIdx int
	previewRequests  []httpfile.Request // requests of collections[collectionIdx], reloaded whenever collectionIdx changes

	// requests/requestIdx/loadedCollection describe whichever request is
	// currently loaded into the [0] Request panel's editor, when it
	// belongs to a collection: requests is that collection's full request
	// list (freshly loaded so ctrl+s can write the whole file back),
	// requestIdx is the loaded request's position within it, and
	// loadedCollection is the owning collection's name. These are
	// independent of collectionIdx/collectionReqIdx, since browsing the
	// Collections panel must not disturb a request already loaded for
	// editing.
	requests         []httpfile.Request
	requestIdx       int
	loadedCollection string

	// scratchRequest is the in-memory request that doesn't belong to any
	// collection; usingScratch reports whether the [0] Request panel's
	// editor currently reflects it (true) or a loaded collection request
	// (false, see above).
	scratchRequest httpfile.Request
	usingScratch   bool

	history    []HistoryEntry
	historyIdx int
	viewingIdx int // -1 = live response; >=0 = viewing history[viewingIdx]

	envNames []string
	envIdx   int

	saveIdx int // selection index within overlaySaveTo

	// savingViaNewCollection is true while overlayNewCollection is
	// servicing a collection-less request's save-to-new-collection flow,
	// as opposed to a plain "create a collection" invocation from the
	// Collections panel.
	savingViaNewCollection bool

	// editor is the single embedded request-editing form, shown inline in
	// the [0] Request panel. It mirrors whichever request is currently
	// loaded (scratchRequest or requests[requestIdx]); edits are synced
	// back on every keystroke (see syncEditorToTarget).
	editor form.Editor

	focus   Panel
	overlay overlay
	input   string // scratch text input for name-prompt overlays

	sending    bool
	cancelSend context.CancelFunc

	statusMsg string
	statusGen int // incremented on every setStatus call; guards against stale auto-clear timers

	width, height int
}

// New constructs a Shell and loads the initial collection list.
func New(colStore *collection.Store, envStore *environment.Store, executor *curlexec.Executor) (*Shell, error) {
	s := &Shell{
		colStore:         colStore,
		envStore:         envStore,
		executor:         executor,
		focus:            PanelRequest,
		viewingIdx:       -1,
		collectionReqIdx: -1,
		usingScratch:     true,
		scratchRequest:   httpfile.Request{Method: "GET"},
	}
	if err := s.reloadCollections(); err != nil {
		return nil, err
	}
	if len(s.collections) > 0 {
		if err := s.reloadCollectionPreview(); err != nil {
			return nil, err
		}
	}
	s.editor = form.FromRequest(s.scratchRequest)
	return s, nil
}

// setFocus moves Shell-level panel focus to p. Unlike the old mode-based
// shell, this never reloads or resets the [0] Request panel's editor --
// whatever request it holds (scratch or a loaded collection request)
// persists across focus changes, since Request is no longer tied 1:1 to
// whichever collection is being browsed in Collections.
func (s *Shell) setFocus(p Panel) {
	s.focus = p
}

// loadRequestIntoEditor loads req (found at index idx within reqs, the
// full freshly-loaded request list of collectionName) into the [0] Request
// panel, binding future ctrl+s/ctrl+r to that collection and index.
func (s *Shell) loadRequestIntoEditor(collectionName string, reqs []httpfile.Request, idx int) {
	s.requests = reqs
	s.requestIdx = idx
	s.loadedCollection = collectionName
	s.usingScratch = false
	s.editor = form.FromRequest(s.requests[idx])
}

// syncEditorToTarget writes the live form state back into the in-memory
// request it reflects (the scratch request, or the loaded collection
// request), keeping it up to date as the user types. Nothing is written to
// disk until ctrl+s.
func (s *Shell) syncEditorToTarget() {
	req := s.editor.ToRequest()
	if s.usingScratch {
		s.scratchRequest = req
		return
	}
	if s.requestIdx >= 0 && s.requestIdx < len(s.requests) {
		s.requests[s.requestIdx] = req
	}
}

func (s *Shell) reloadCollections() error {
	cols, err := s.colStore.List()
	if err != nil {
		return err
	}
	s.collections = cols
	if s.collectionIdx >= len(cols) {
		s.collectionIdx = max0(len(cols) - 1)
	}
	return nil
}

func (s *Shell) currentCollectionName() string {
	if s.collectionIdx < 0 || s.collectionIdx >= len(s.collections) {
		return ""
	}
	return s.collections[s.collectionIdx].Name
}

// reloadCollectionPreview (re)loads previewRequests from whichever
// collection the Collections panel cursor rests on (collectionIdx), for
// accordion rendering and duplicate/delete/new-request actions. It also
// refreshes the environment list, which is likewise scoped to the browsed
// collection.
func (s *Shell) reloadCollectionPreview() error {
	name := s.currentCollectionName()
	if name == "" {
		s.previewRequests = nil
		s.collectionReqIdx = -1
		s.envNames = nil
		return nil
	}
	reqs, err := s.colStore.LoadRequests(name)
	if err != nil {
		return err
	}
	s.previewRequests = reqs
	if s.collectionReqIdx >= len(reqs) {
		s.collectionReqIdx = len(reqs) - 1
	}
	return s.reloadEnvironments()
}

func (s *Shell) reloadEnvironments() error {
	name := s.currentCollectionName()
	if name == "" {
		s.envNames = nil
		return nil
	}
	names, err := s.envStore.List(name)
	if err != nil {
		return err
	}
	s.envNames = names

	active, err := s.envStore.ActiveEnvironment(name)
	if err != nil {
		return err
	}
	s.envIdx = 0
	for i, n := range names {
		if n == active {
			s.envIdx = i
			break
		}
	}
	return nil
}

func (s *Shell) activeEnvName() string {
	if s.envIdx < 0 || s.envIdx >= len(s.envNames) {
		return ""
	}
	return s.envNames[s.envIdx]
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}

// Collections returns the currently loaded collection list.
func (s *Shell) Collections() []collection.Collection { return s.collections }

// History returns the executed request/response history, oldest first.
func (s *Shell) History() []HistoryEntry { return s.history }

// ScratchRequest returns the in-memory request that doesn't belong to any
// collection.
func (s *Shell) ScratchRequest() httpfile.Request { return s.scratchRequest }

// SetScratchRequest replaces the in-memory scratch request directly (used
// by tests as a convenience seam; normal editing goes through the embedded
// form and syncEditorToTarget instead). It never touches disk -- the
// request stays unsaved until the user saves it to a collection.
func (s *Shell) SetScratchRequest(req httpfile.Request) {
	s.scratchRequest = req
	if s.usingScratch {
		s.editor = form.FromRequest(req)
	}
}

// Init satisfies tea.Model; Shell has no async startup work.
func (s *Shell) Init() tea.Cmd { return nil }

// SetSize updates the shell's rendering dimensions.
func (s *Shell) SetSize(w, h int) {
	s.width, s.height = w, h
}

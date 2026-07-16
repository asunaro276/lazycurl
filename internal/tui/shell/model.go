// Package shell implements lazycurl's TUI shell: the collection/request/
// response/history panel layout, lazygit-compatible keybindings, and
// colored status/method badges.
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

// Mode identifies which top-level screen the shell shows: the
// collection-free Adhoc request builder, or the full Collections browser.
type Mode int

const (
	ModeAdhoc Mode = iota
	ModeCollections
)

// Panel identifies one of the shell's navigable panels.
type Panel int

const (
	PanelCollections Panel = iota
	PanelRequests
	PanelResponse
	PanelHistory
	PanelEditor // Adhoc mode's request-editor panel
)

var panelLabels = map[Panel]string{
	PanelCollections: "Collections",
	PanelRequests:    "Requests",
	PanelResponse:    "Response",
	PanelHistory:     "History",
	PanelEditor:      "Editor",
}

// requestZone identifies which part of the Collections mode Requests panel
// currently has focus: the request list, or the embedded editor form for
// the selected request. Adhoc mode's Editor panel has no list zone -- it is
// always in the form zone once focused.
type requestZone int

const (
	zoneList requestZone = iota
	zoneForm
)

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
	overlaySaveAdhoc
	overlayRequestName
)

// QuitMsg is emitted when the user requests to quit.
type QuitMsg struct{}

// Shell is the main panel-based TUI: Collections, Requests, Response, and
// History panels, navigated with lazygit-style keybindings.
type Shell struct {
	colStore *collection.Store
	envStore *environment.Store
	executor *curlexec.Executor

	collections   []collection.Collection
	collectionIdx int

	requests   []httpfile.Request
	requestIdx int

	history    []HistoryEntry
	historyIdx int
	viewingIdx int // -1 = live response; >=0 = viewing history[viewingIdx]

	envNames []string
	envIdx   int

	mode         Mode
	adhocRequest httpfile.Request // Adhoc mode's unsaved scratch request
	saveIdx      int              // selection index within overlaySaveAdhoc
	savingAdhoc  bool             // true while overlayNewCollection is servicing an Adhoc save
	namingAdhoc  bool             // true while overlayRequestName is servicing an Adhoc save (vs. a Collections save)

	// editor is the embedded request-editing form shown inline in the
	// Requests panel (Collections, when reqZone==zoneForm) or the Editor
	// panel (Adhoc, always). It mirrors whichever request is currently
	// selected; edits are synced back on every keystroke (see
	// syncEditorToTarget).
	editor  form.Editor
	reqZone requestZone // Collections' Requests panel: zoneList or zoneForm

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
		colStore:     colStore,
		envStore:     envStore,
		executor:     executor,
		mode:         ModeAdhoc,
		focus:        PanelEditor,
		viewingIdx:   -1,
		adhocRequest: httpfile.Request{Method: "GET"},
	}
	if err := s.reloadCollections(); err != nil {
		return nil, err
	}
	if len(s.collections) > 0 {
		if err := s.loadRequestsForCurrentCollection(); err != nil {
			return nil, err
		}
	}
	s.editor = form.FromRequest(s.adhocRequest)
	return s, nil
}

// setFocus moves Shell-level panel focus to p, resetting per-panel
// sub-focus state so the Requests/Editor panel always (re-)enters at its
// default starting point: the request list (Collections) or the embedded
// form loaded fresh from its target request (Adhoc, and Collections once
// the form zone is (re)entered explicitly via loadEditorForCurrentTarget).
func (s *Shell) setFocus(p Panel) {
	s.focus = p
	switch p {
	case PanelRequests:
		s.reqZone = zoneList
	case PanelEditor:
		s.loadEditorForCurrentTarget()
	}
}

// inFormZone reports whether the Requests/Editor panel's embedded form
// currently owns keyboard focus: always true for Adhoc's Editor panel
// (which has no list zone), true for Collections' Requests panel only once
// its form zone has been entered.
func (s *Shell) inFormZone() bool {
	switch s.focus {
	case PanelEditor:
		return true
	case PanelRequests:
		return s.reqZone == zoneForm
	}
	return false
}

// loadEditorForCurrentTarget (re)loads s.editor from whichever request it
// should currently reflect: the Adhoc scratch request, or the selected
// Collections request. The reloaded form always starts at its first field.
func (s *Shell) loadEditorForCurrentTarget() {
	if s.mode == ModeAdhoc {
		s.editor = form.FromRequest(s.adhocRequest)
		return
	}
	if s.requestIdx >= 0 && s.requestIdx < len(s.requests) {
		s.editor = form.FromRequest(s.requests[s.requestIdx])
	}
}

// syncEditorToTarget writes the live form state back into the in-memory
// request it reflects, keeping the Requests list summary and eventual
// ctrl+s disk saves up to date as the user types.
func (s *Shell) syncEditorToTarget() {
	req := s.editor.ToRequest()
	if s.mode == ModeAdhoc {
		s.adhocRequest = req
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

func (s *Shell) loadRequestsForCurrentCollection() error {
	name := s.currentCollectionName()
	if name == "" {
		s.requests = nil
		s.envNames = nil
		return nil
	}
	reqs, err := s.colStore.LoadRequests(name)
	if err != nil {
		return err
	}
	s.requests = reqs
	if s.requestIdx >= len(reqs) {
		s.requestIdx = max0(len(reqs) - 1)
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

// Requests returns the requests of the currently selected collection.
func (s *Shell) Requests() []httpfile.Request { return s.requests }

// History returns the executed request/response history, oldest first.
func (s *Shell) History() []HistoryEntry { return s.history }

// Mode returns the shell's current top-level mode (Adhoc or Collections).
func (s *Shell) Mode() Mode { return s.mode }

// AdhocRequest returns the in-memory Adhoc scratch request.
func (s *Shell) AdhocRequest() httpfile.Request { return s.adhocRequest }

// SetAdhocRequest replaces the in-memory Adhoc scratch request directly
// (used by tests as a convenience seam; normal editing goes through the
// embedded form and syncEditorToTarget instead). It never touches disk --
// the request stays unsaved until the user saves it to a collection.
func (s *Shell) SetAdhocRequest(req httpfile.Request) {
	s.adhocRequest = req
}

// Init satisfies tea.Model; Shell has no async startup work.
func (s *Shell) Init() tea.Cmd { return nil }

// SetSize updates the shell's rendering dimensions.
func (s *Shell) SetSize(w, h int) {
	s.width, s.height = w, h
}

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
)

// Panel identifies one of the shell's navigable panels.
type Panel int

const (
	PanelCollections Panel = iota
	PanelRequests
	PanelResponse
	PanelHistory
	PanelEdit // Adhoc mode only: the scratch request edit pane
)

var panelLabels = map[Panel]string{
	PanelCollections: "Collections",
	PanelRequests:    "Requests",
	PanelResponse:    "Response",
	PanelHistory:     "History",
	PanelEdit:        "Request",
}

// Mode identifies which top-level UI mode the shell is displaying.
type Mode int

const (
	ModeAdhoc Mode = iota
	ModeCollections
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
	overlaySaveTarget
)

// OpenEditorMsg is emitted by Shell when the user requests creating or
// editing a request; the parent App owns the form.Editor and handles this.
type OpenEditorMsg struct {
	CollectionName string
	Request        httpfile.Request
	Index          int // -1 for a new request
}

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
	adhocRequest httpfile.Request // Adhoc mode's in-memory scratch request

	focus   Panel
	overlay overlay
	input   string // scratch text input for name-prompt overlays

	saveOverlayIdx   int  // selection index within overlaySaveTarget
	pendingAdhocSave bool // true while overlayNewCollection is being used to pick an Adhoc save target

	sending    bool
	cancelSend context.CancelFunc

	statusMsg string

	width, height int
}

// New constructs a Shell and loads the initial collection list.
func New(colStore *collection.Store, envStore *environment.Store, executor *curlexec.Executor) (*Shell, error) {
	s := &Shell{
		colStore:     colStore,
		envStore:     envStore,
		executor:     executor,
		mode:         ModeAdhoc,
		adhocRequest: httpfile.Request{Method: "GET"},
		focus:        PanelEdit,
		viewingIdx:   -1,
	}
	if err := s.reloadCollections(); err != nil {
		return nil, err
	}
	if len(s.collections) > 0 {
		if err := s.loadRequestsForCurrentCollection(); err != nil {
			return nil, err
		}
	}
	return s, nil
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

// Mode returns the shell's current top-level UI mode.
func (s *Shell) Mode() Mode { return s.mode }

// AdhocRequest returns the current in-memory Adhoc scratch request.
func (s *Shell) AdhocRequest() httpfile.Request { return s.adhocRequest }

// UpdateAdhocRequest replaces the in-memory Adhoc scratch request. Called by
// the parent App after a form save when editing outside any collection.
func (s *Shell) UpdateAdhocRequest(req httpfile.Request) {
	s.adhocRequest = req
}

// ReloadCurrentCollection re-reads the selected collection's requests from
// disk. Called by the parent App after a form save changes the underlying
// .http file out from under the shell.
func (s *Shell) ReloadCurrentCollection() error {
	return s.loadRequestsForCurrentCollection()
}

// Init satisfies tea.Model; Shell has no async startup work.
func (s *Shell) Init() tea.Cmd { return nil }

// SetSize updates the shell's rendering dimensions.
func (s *Shell) SetSize(w, h int) {
	s.width, s.height = w, h
}

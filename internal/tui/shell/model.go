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
	adhocRequest httpfile.Request // Adhoc mode's unsaved scratch request
	saveIdx      int              // selection index within overlaySaveAdhoc
	savingAdhoc  bool             // true while overlayNewCollection is servicing an Adhoc save

	focus   Panel
	overlay overlay
	input   string // scratch text input for name-prompt overlays

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

// Mode returns the shell's current top-level mode (Adhoc or Collections).
func (s *Shell) Mode() Mode { return s.mode }

// AdhocRequest returns the in-memory Adhoc scratch request.
func (s *Shell) AdhocRequest() httpfile.Request { return s.adhocRequest }

// SetAdhocRequest replaces the in-memory Adhoc scratch request. Called by
// the parent App after a form save while editing in Adhoc mode; it never
// touches disk (the request stays unsaved until the user saves it to a
// collection from Adhoc mode).
func (s *Shell) SetAdhocRequest(req httpfile.Request) {
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

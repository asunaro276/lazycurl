package shell

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/httpfile"
)

type sendResultMsg struct {
	entry HistoryEntry
}

// Update handles all shell input: panel navigation, sending requests,
// overlays (help/env-select/new-collection/delete-confirm), and quitting.
func (s *Shell) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.SetSize(msg.Width, msg.Height)
		return nil
	case sendResultMsg:
		s.sending = false
		s.cancelSend = nil
		s.history = append(s.history, msg.entry)
		s.historyIdx = len(s.history) - 1
		s.viewingIdx = -1
		if msg.entry.Err != nil {
			s.statusMsg = msg.entry.Err.Error()
		} else {
			s.statusMsg = ""
		}
		return nil
	case tea.KeyMsg:
		return s.handleKey(msg)
	}
	return nil
}

func (s *Shell) handleKey(msg tea.KeyMsg) tea.Cmd {
	if s.overlay != overlayNone {
		return s.handleOverlayKey(msg)
	}

	switch msg.String() {
	case "ctrl+c":
		if s.sending {
			s.cancel()
			return nil
		}
		return func() tea.Msg { return QuitMsg{} }
	case "q":
		return func() tea.Msg { return QuitMsg{} }
	case "?":
		s.overlay = overlayHelp
		return nil
	case "[", "]":
		s.toggleMode()
		return nil
	case "s":
		if s.mode == ModeAdhoc {
			s.overlay = overlaySaveAdhoc
			s.saveIdx = 0
			return nil
		}
	case "tab":
		s.focusNext()
		return nil
	case "shift+tab":
		s.focusPrev()
		return nil
	case "1", "2", "3", "4":
		panels := s.panelsForMode()
		n := int(msg.String()[0] - '1')
		if n < len(panels) {
			s.focus = panels[n]
		}
		return nil
	}

	switch s.focus {
	case PanelCollections:
		return s.handleCollectionsKey(msg)
	case PanelRequests:
		return s.handleRequestsKey(msg)
	case PanelHistory:
		return s.handleHistoryKey(msg)
	case PanelResponse:
		return s.handleResponseKey(msg)
	case PanelEditor:
		return s.handleEditorPanelKey(msg)
	}
	return nil
}

// panelsForMode returns the ordered panel set navigable in the shell's
// current mode: the 3-pane Adhoc layout, or the 4-pane Collections layout.
func (s *Shell) panelsForMode() []Panel {
	if s.mode == ModeAdhoc {
		return []Panel{PanelEditor, PanelResponse, PanelHistory}
	}
	return []Panel{PanelCollections, PanelRequests, PanelResponse, PanelHistory}
}

func panelIndex(panels []Panel, p Panel) (int, bool) {
	for i, x := range panels {
		if x == p {
			return i, true
		}
	}
	return 0, false
}

func (s *Shell) focusNext() {
	panels := s.panelsForMode()
	idx, _ := panelIndex(panels, s.focus)
	s.focus = panels[(idx+1)%len(panels)]
}

func (s *Shell) focusPrev() {
	panels := s.panelsForMode()
	idx, _ := panelIndex(panels, s.focus)
	s.focus = panels[(idx-1+len(panels))%len(panels)]
}

// toggleMode switches between Adhoc and Collections, closing any open
// overlay and resetting focus to the new mode's first panel unless the
// current focus (e.g. Response/History) is still valid there.
func (s *Shell) toggleMode() {
	if s.mode == ModeAdhoc {
		s.mode = ModeCollections
	} else {
		s.mode = ModeAdhoc
	}
	panels := s.panelsForMode()
	if _, ok := panelIndex(panels, s.focus); !ok {
		s.focus = panels[0]
	}
	s.overlay = overlayNone
}

// handleEditorPanelKey handles input while Adhoc mode's Editor panel is
// focused: 'e' opens the request-editor form (via OpenEditorMsg, reusing the
// same App-owned form as Collections mode), 'enter' sends the scratch
// request as-is.
func (s *Shell) handleEditorPanelKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "e":
		req := s.adhocRequest
		return func() tea.Msg {
			return OpenEditorMsg{CollectionName: "", Request: req, Index: -1}
		}
	case "enter":
		return s.sendAdhocCurrent()
	}
	return nil
}

func (s *Shell) handleCollectionsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if s.collectionIdx < len(s.collections)-1 {
			s.collectionIdx++
			s.requestIdx = 0
			s.statusMsg = ""
			if err := s.loadRequestsForCurrentCollection(); err != nil {
				s.statusMsg = err.Error()
			}
		}
	case "k", "up":
		if s.collectionIdx > 0 {
			s.collectionIdx--
			s.requestIdx = 0
			s.statusMsg = ""
			if err := s.loadRequestsForCurrentCollection(); err != nil {
				s.statusMsg = err.Error()
			}
		}
	case "enter":
		s.focus = PanelRequests
	case "n":
		s.overlay = overlayNewCollection
		s.input = ""
	case "E":
		if s.currentCollectionName() != "" {
			s.overlay = overlayEnvSelect
		}
	}
	return nil
}

func (s *Shell) handleRequestsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if s.requestIdx < len(s.requests)-1 {
			s.requestIdx++
		}
	case "k", "up":
		if s.requestIdx > 0 {
			s.requestIdx--
		}
	case "enter":
		return s.sendCurrent()
	case "e":
		if s.requestIdx < len(s.requests) {
			req := s.requests[s.requestIdx]
			idx := s.requestIdx
			return func() tea.Msg {
				return OpenEditorMsg{CollectionName: s.currentCollectionName(), Request: req, Index: idx}
			}
		}
	case "n":
		if s.currentCollectionName() == "" {
			s.statusMsg = "先にコレクションを作成してください"
			return nil
		}
		return func() tea.Msg {
			return OpenEditorMsg{CollectionName: s.currentCollectionName(), Request: httpfile.Request{Method: "GET"}, Index: -1}
		}
	case "c":
		if s.requestIdx < len(s.requests) {
			if err := s.colStore.DuplicateRequest(s.currentCollectionName(), s.requestIdx); err != nil {
				s.statusMsg = err.Error()
				return nil
			}
			if err := s.loadRequestsForCurrentCollection(); err != nil {
				s.statusMsg = err.Error()
			}
		}
	case "d", "x":
		if s.requestIdx < len(s.requests) {
			s.overlay = overlayConfirmDelete
		}
	case "E":
		if s.currentCollectionName() != "" {
			s.overlay = overlayEnvSelect
		}
	}
	return nil
}

func (s *Shell) handleHistoryKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if s.historyIdx < len(s.history)-1 {
			s.historyIdx++
		}
	case "k", "up":
		if s.historyIdx > 0 {
			s.historyIdx--
		}
	case "enter":
		if s.historyIdx < len(s.history) {
			s.viewingIdx = s.historyIdx
			s.focus = PanelResponse
		}
	}
	return nil
}

func (s *Shell) handleResponseKey(msg tea.KeyMsg) tea.Cmd {
	return nil
}

func (s *Shell) handleOverlayKey(msg tea.KeyMsg) tea.Cmd {
	switch s.overlay {
	case overlayHelp:
		switch msg.String() {
		case "?", "esc", "q", "enter":
			s.overlay = overlayNone
		}
		return nil

	case overlayEnvSelect:
		switch msg.String() {
		case "esc", "q":
			s.overlay = overlayNone
		case "j", "down":
			if s.envIdx < len(s.envNames)-1 {
				s.envIdx++
			}
		case "k", "up":
			if s.envIdx > 0 {
				s.envIdx--
			}
		case "enter":
			name := s.currentCollectionName()
			if name != "" && s.envIdx < len(s.envNames) {
				if err := s.envStore.SetActiveEnvironment(name, s.envNames[s.envIdx]); err != nil {
					s.statusMsg = err.Error()
				}
			}
			s.overlay = overlayNone
		}
		return nil

	case overlayNewCollection:
		switch msg.String() {
		case "esc":
			s.overlay = overlayNone
			s.savingAdhoc = false
		case "enter":
			name := strings.TrimSpace(s.input)
			if name != "" {
				if err := s.colStore.CreateCollection(name); err != nil {
					s.statusMsg = err.Error()
				} else if err := s.reloadCollections(); err != nil {
					s.statusMsg = err.Error()
				} else {
					for i, c := range s.collections {
						if c.Name == name {
							s.collectionIdx = i
						}
					}
					if s.savingAdhoc {
						s.finishAdhocSave(name)
					} else {
						_ = s.loadRequestsForCurrentCollection()
					}
				}
			}
			s.savingAdhoc = false
			s.overlay = overlayNone
		case "backspace":
			if len(s.input) > 0 {
				s.input = s.input[:len(s.input)-1]
			}
		default:
			if msg.Type == tea.KeyRunes {
				s.input += string(msg.Runes)
			}
		}
		return nil

	case overlaySaveAdhoc:
		switch msg.String() {
		case "esc", "q":
			s.overlay = overlayNone
		case "j", "down":
			if s.saveIdx < len(s.collections) {
				s.saveIdx++
			}
		case "k", "up":
			if s.saveIdx > 0 {
				s.saveIdx--
			}
		case "enter":
			if s.saveIdx == len(s.collections) {
				s.savingAdhoc = true
				s.overlay = overlayNewCollection
				s.input = ""
				return nil
			}
			if s.saveIdx < len(s.collections) {
				s.finishAdhocSave(s.collections[s.saveIdx].Name)
			}
			s.overlay = overlayNone
		}
		return nil

	case overlayConfirmDelete:
		switch msg.String() {
		case "y":
			if err := s.colStore.DeleteRequest(s.currentCollectionName(), s.requestIdx); err != nil {
				s.statusMsg = err.Error()
			} else if err := s.loadRequestsForCurrentCollection(); err != nil {
				s.statusMsg = err.Error()
			}
			s.overlay = overlayNone
		case "n", "esc":
			s.overlay = overlayNone
		}
		return nil
	}
	return nil
}

// finishAdhocSave appends the Adhoc scratch request to collectionName's
// `.http` file, then switches to Collections mode with that collection and
// the newly-saved request selected, and resets the scratch request.
func (s *Shell) finishAdhocSave(collectionName string) {
	if err := s.colStore.CreateRequest(collectionName, s.adhocRequest); err != nil {
		s.statusMsg = err.Error()
		return
	}
	for i, c := range s.collections {
		if c.Name == collectionName {
			s.collectionIdx = i
		}
	}
	if err := s.loadRequestsForCurrentCollection(); err != nil {
		s.statusMsg = err.Error()
		return
	}
	s.requestIdx = max0(len(s.requests) - 1)
	s.adhocRequest = httpfile.Request{Method: "GET"}
	s.mode = ModeCollections
	s.focus = PanelRequests
}

func (s *Shell) cancel() {
	if s.cancelSend != nil {
		s.cancelSend()
	}
}

// sendCurrent expands the selected request's variables against the active
// environment and, if fully defined, executes it via curl asynchronously.
func (s *Shell) sendCurrent() tea.Cmd {
	if s.sending || s.requestIdx >= len(s.requests) {
		return nil
	}
	collectionName := s.currentCollectionName()
	req := s.requests[s.requestIdx]

	vars := map[string]string{}
	if envName := s.activeEnvName(); envName != "" {
		v, err := s.envStore.Load(collectionName, envName)
		if err != nil {
			s.statusMsg = err.Error()
			return nil
		}
		vars = v
	}

	expanded, undefined := environment.ExpandRequest(req, vars)
	if len(undefined) > 0 {
		s.statusMsg = fmt.Sprintf("未定義の変数があります: %s", strings.Join(undefined, ", "))
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelSend = cancel
	s.sending = true
	s.statusMsg = ""

	executor := s.executor
	return func() tea.Msg {
		resp, err := executor.Execute(ctx, expanded)
		return sendResultMsg{entry: HistoryEntry{
			CollectionName: collectionName,
			Request:        expanded,
			Response:       resp,
			Err:            err,
			At:             time.Now(),
		}}
	}
}

// sendAdhocCurrent executes the Adhoc scratch request as-is, with no
// {{variable}} expansion (Adhoc requests aren't tied to a collection or
// environment).
func (s *Shell) sendAdhocCurrent() tea.Cmd {
	if s.sending {
		return nil
	}
	req := s.adhocRequest
	if strings.TrimSpace(req.URL) == "" {
		s.statusMsg = "URLを入力してください"
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelSend = cancel
	s.sending = true
	s.statusMsg = ""

	executor := s.executor
	return func() tea.Msg {
		resp, err := executor.Execute(ctx, req)
		return sendResultMsg{entry: HistoryEntry{
			CollectionName: "",
			Request:        req,
			Response:       resp,
			Err:            err,
			At:             time.Now(),
		}}
	}
}

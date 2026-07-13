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

// clearStatusMsg requests clearing the status bar, but only if statusGen
// still matches the generation at schedule time (i.e. no newer status
// message has replaced it since).
type clearStatusMsg struct {
	gen int
}

// setStatus sets the status bar message and, for non-empty messages,
// schedules it to auto-clear after 5 seconds. Every call bumps statusGen,
// so a stale clearStatusMsg from an earlier call is a no-op once a newer
// message has replaced it.
func (s *Shell) setStatus(msg string) tea.Cmd {
	s.statusMsg = msg
	s.statusGen++
	if msg == "" {
		return nil
	}
	gen := s.statusGen
	return tea.Tick(5*time.Second, func(time.Time) tea.Msg {
		return clearStatusMsg{gen: gen}
	})
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
			return s.setStatus(msg.entry.Err.Error())
		}
		return s.setStatus("")
	case clearStatusMsg:
		if msg.gen == s.statusGen {
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
	if s.inFormZone() {
		return s.handleFormZoneKey(msg)
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
	case "tab":
		if s.focus == PanelRequests && s.reqZone == zoneList && s.requestIdx < len(s.requests) {
			s.reqZone = zoneForm
			s.loadEditorForCurrentTarget()
			return nil
		}
		s.focusNext()
		return nil
	case "shift+tab":
		s.focusPrev()
		return nil
	case "1", "2", "3", "4":
		panels := s.panelsForMode()
		n := int(msg.String()[0] - '1')
		if n < len(panels) {
			s.setFocus(panels[n])
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
	s.setFocus(panels[(idx+1)%len(panels)])
}

func (s *Shell) focusPrev() {
	panels := s.panelsForMode()
	idx, _ := panelIndex(panels, s.focus)
	s.setFocus(panels[(idx-1+len(panels))%len(panels)])
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
		s.setFocus(panels[0])
	}
	s.overlay = overlayNone
}

// handleFormZoneKey routes keys to the embedded form.Editor while the
// Requests/Editor panel's form zone has focus, bypassing the global
// shortcuts handled above (which would otherwise steal characters meant
// for text fields). Only ctrl+c stays global here; ctrl+s (save) and
// ctrl+r (send) replace the bare `s`/`enter` bindings used outside the
// form zone, since `enter` is needed for in-field editing (e.g. Body
// newlines).
func (s *Shell) handleFormZoneKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		if s.sending {
			s.cancel()
			return nil
		}
		return func() tea.Msg { return QuitMsg{} }
	case "ctrl+s":
		return s.saveFormZone()
	case "ctrl+r":
		return s.sendFormZone()
	case "tab":
		if s.editor.AtLastFocus() {
			s.exitFormZoneForward()
			return nil
		}
	case "shift+tab":
		if s.editor.AtFirstFocus() {
			s.exitFormZoneBackward()
			return nil
		}
	}

	var cmd tea.Cmd
	s.editor, cmd = s.editor.Update(msg)
	s.syncEditorToTarget()
	return cmd
}

// exitFormZoneForward moves focus out of the form (past its last field) to
// the next Shell panel, returning Collections' Requests panel to its list
// zone for next time.
func (s *Shell) exitFormZoneForward() {
	if s.focus == PanelRequests {
		s.reqZone = zoneList
	}
	s.focusNext()
}

// exitFormZoneBackward moves focus out of the form (before its first
// field): back to the request list for Collections, or to the previous
// Shell panel for Adhoc (which has no list zone).
func (s *Shell) exitFormZoneBackward() {
	if s.focus == PanelRequests {
		s.reqZone = zoneList
		return
	}
	s.focusPrev()
}

// saveFormZone persists the form's target request: to the selected
// collection's `.http` file (Collections), or via the save-to-collection
// overlay (Adhoc, which has no standalone file to write to).
func (s *Shell) saveFormZone() tea.Cmd {
	if s.mode == ModeAdhoc {
		s.overlay = overlaySaveAdhoc
		s.saveIdx = 0
		return nil
	}
	if err := s.colStore.SaveRequests(s.currentCollectionName(), s.requests); err != nil {
		return s.setStatus(err.Error())
	}
	return s.setStatus("")
}

// sendFormZone sends the form's target request: the selected Collections
// request (with variable expansion), or the Adhoc scratch request as-is.
func (s *Shell) sendFormZone() tea.Cmd {
	if s.mode == ModeAdhoc {
		return s.sendAdhocCurrent()
	}
	return s.sendCurrent()
}

func (s *Shell) handleCollectionsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		if s.collectionIdx < len(s.collections)-1 {
			s.collectionIdx++
			s.requestIdx = 0
			if err := s.loadRequestsForCurrentCollection(); err != nil {
				return s.setStatus(err.Error())
			}
			return s.setStatus("")
		}
	case "k", "up":
		if s.collectionIdx > 0 {
			s.collectionIdx--
			s.requestIdx = 0
			if err := s.loadRequestsForCurrentCollection(); err != nil {
				return s.setStatus(err.Error())
			}
			return s.setStatus("")
		}
	case "enter":
		s.setFocus(PanelRequests)
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
	case "n":
		if s.currentCollectionName() == "" {
			return s.setStatus("先にコレクションを作成してください")
		}
		s.requests = append(s.requests, httpfile.Request{Method: "GET"})
		s.requestIdx = len(s.requests) - 1
		s.reqZone = zoneForm
		s.loadEditorForCurrentTarget()
		return nil
	case "c":
		if s.requestIdx < len(s.requests) {
			if err := s.colStore.DuplicateRequest(s.currentCollectionName(), s.requestIdx); err != nil {
				return s.setStatus(err.Error())
			}
			if err := s.loadRequestsForCurrentCollection(); err != nil {
				return s.setStatus(err.Error())
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
			s.setFocus(PanelResponse)
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
			var cmd tea.Cmd
			if name != "" && s.envIdx < len(s.envNames) {
				if err := s.envStore.SetActiveEnvironment(name, s.envNames[s.envIdx]); err != nil {
					cmd = s.setStatus(err.Error())
				}
			}
			s.overlay = overlayNone
			return cmd
		}
		return nil

	case overlayNewCollection:
		switch msg.String() {
		case "esc":
			s.overlay = overlayNone
			s.savingAdhoc = false
		case "enter":
			name := strings.TrimSpace(s.input)
			var cmd tea.Cmd
			if name != "" {
				if err := s.colStore.CreateCollection(name); err != nil {
					cmd = s.setStatus(err.Error())
				} else if err := s.reloadCollections(); err != nil {
					cmd = s.setStatus(err.Error())
				} else {
					for i, c := range s.collections {
						if c.Name == name {
							s.collectionIdx = i
						}
					}
					if s.savingAdhoc {
						cmd = s.finishAdhocSave(name)
					} else {
						_ = s.loadRequestsForCurrentCollection()
					}
				}
			}
			s.savingAdhoc = false
			s.overlay = overlayNone
			return cmd
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
				cmd := s.finishAdhocSave(s.collections[s.saveIdx].Name)
				s.overlay = overlayNone
				return cmd
			}
			s.overlay = overlayNone
		}
		return nil

	case overlayConfirmDelete:
		switch msg.String() {
		case "y":
			var cmd tea.Cmd
			if err := s.colStore.DeleteRequest(s.currentCollectionName(), s.requestIdx); err != nil {
				cmd = s.setStatus(err.Error())
			} else if err := s.loadRequestsForCurrentCollection(); err != nil {
				cmd = s.setStatus(err.Error())
			}
			s.overlay = overlayNone
			return cmd
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
func (s *Shell) finishAdhocSave(collectionName string) tea.Cmd {
	if err := s.colStore.CreateRequest(collectionName, s.adhocRequest); err != nil {
		return s.setStatus(err.Error())
	}
	for i, c := range s.collections {
		if c.Name == collectionName {
			s.collectionIdx = i
		}
	}
	if err := s.loadRequestsForCurrentCollection(); err != nil {
		return s.setStatus(err.Error())
	}
	s.requestIdx = max0(len(s.requests) - 1)
	s.adhocRequest = httpfile.Request{Method: "GET"}
	s.mode = ModeCollections
	s.setFocus(PanelRequests)
	return nil
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
			return s.setStatus(err.Error())
		}
		vars = v
	}

	expanded, undefined := environment.ExpandRequest(req, vars)
	if len(undefined) > 0 {
		return s.setStatus(fmt.Sprintf("未定義の変数があります: %s", strings.Join(undefined, ", ")))
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelSend = cancel
	s.sending = true
	s.setStatus("")

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
		return s.setStatus("URLを入力してください")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancelSend = cancel
	s.sending = true
	s.setStatus("")

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

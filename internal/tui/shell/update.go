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
	if msg.String() == "ctrl+c" {
		if s.sending {
			s.cancel()
			return nil
		}
		return func() tea.Msg { return QuitMsg{} }
	}

	if s.focus == PanelRequest {
		return s.handleRequestKey(msg)
	}

	if cmd, ok := s.handleGlobalKey(msg); ok {
		return cmd
	}

	switch s.focus {
	case PanelCollections:
		return s.handleCollectionsKey(msg)
	case PanelHistory:
		return s.handleHistoryKey(msg)
	case PanelResponse:
		return s.handleResponseKey(msg)
	}
	return nil
}

// handleGlobalKey handles shortcuts available whenever no field is
// capturing keystrokes: quit, help, and panel switching (tab/shift+tab/
// 0-3). Returns ok=false if msg isn't one of these, so the caller can fall
// through to panel-specific handling.
func (s *Shell) handleGlobalKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case "q":
		return func() tea.Msg { return QuitMsg{} }, true
	case "?":
		s.overlay = overlayHelp
		return nil, true
	case "tab":
		s.focusNext()
		return nil, true
	case "shift+tab":
		s.focusPrev()
		return nil, true
	case "0", "1", "2", "3":
		s.setFocus(Panel(msg.String()[0] - '0'))
		return nil, true
	}
	return nil, false
}

func (s *Shell) focusNext() { s.setFocus(Panel((int(s.focus) + 1) % panelCount)) }
func (s *Shell) focusPrev() { s.setFocus(Panel((int(s.focus) + panelCount - 1) % panelCount)) }

// handleRequestKey handles all keys while the [0] Request panel has focus,
// covering both its normal and insert (editor.Editing()) states. ctrl+s
// (save) and ctrl+r (send) are available in both; panel-switch shortcuts
// (0-3, tab, shift+tab, q, ?) only fire in normal state -- once the editor
// is in insert, every other key is forwarded to it so it can be typed
// literally.
func (s *Shell) handleRequestKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+s":
		return s.saveRequestPanel()
	case "ctrl+r":
		return s.sendRequestPanel()
	}

	if !s.editor.Editing() {
		if cmd, ok := s.handleGlobalKey(msg); ok {
			return cmd
		}
	}

	var cmd tea.Cmd
	s.editor, cmd = s.editor.Update(msg)
	s.syncEditorToTarget()
	return cmd
}

// saveRequestPanel persists the [0] Request panel's target request: to the
// selected collection's `.http` file (if it's already bound to one), or via
// the save-to-collection overlay (if it's the collection-less scratch
// request). If the target has no name yet, it opens overlayRequestName to
// collect one first and defers the actual save until the name is
// confirmed; a request that already has a name skips the prompt.
func (s *Shell) saveRequestPanel() tea.Cmd {
	if s.usingScratch {
		if strings.TrimSpace(s.scratchRequest.Name) == "" {
			s.overlay = overlayRequestName
			s.input = ""
			return nil
		}
		return s.beginScratchSave()
	}
	if s.requestIdx < len(s.requests) && strings.TrimSpace(s.requests[s.requestIdx].Name) == "" {
		s.overlay = overlayRequestName
		s.input = ""
		return nil
	}
	return s.saveLoadedRequests()
}

// beginScratchSave opens the save-to-collection overlay for the (now-named)
// scratch request.
func (s *Shell) beginScratchSave() tea.Cmd {
	s.overlay = overlaySaveTo
	s.saveIdx = 0
	return nil
}

// saveLoadedRequests writes the loaded collection's in-memory requests to
// its `.http` file.
func (s *Shell) saveLoadedRequests() tea.Cmd {
	if err := s.colStore.SaveRequests(s.loadedCollection, s.requests); err != nil {
		return s.setStatus(err.Error())
	}
	return s.setStatus("")
}

// sendRequestPanel sends the [0] Request panel's target request: the loaded
// collection request (with variable expansion), or the scratch request as-is.
func (s *Shell) sendRequestPanel() tea.Cmd {
	if s.usingScratch {
		return s.sendScratchCurrent()
	}
	return s.sendLoadedCurrent()
}

// collectionsCursorDown moves the Collections panel's flattened cursor one
// row down: from a collection's header into its first request row, through
// its request rows, then on to the next collection's header (which becomes
// the newly-expanded accordion).
func (s *Shell) collectionsCursorDown() {
	if s.collectionReqIdx == -1 {
		if len(s.previewRequests) > 0 {
			s.collectionReqIdx = 0
			return
		}
	} else if s.collectionReqIdx < len(s.previewRequests)-1 {
		s.collectionReqIdx++
		return
	}
	if s.collectionIdx < len(s.collections)-1 {
		s.collectionIdx++
		s.collectionReqIdx = -1
		_ = s.reloadCollectionPreview()
	}
}

// collectionsCursorUp is the reverse of collectionsCursorDown.
func (s *Shell) collectionsCursorUp() {
	if s.collectionReqIdx > 0 {
		s.collectionReqIdx--
		return
	}
	if s.collectionReqIdx == -1 {
		if s.collectionIdx > 0 {
			s.collectionIdx--
			_ = s.reloadCollectionPreview()
			s.collectionReqIdx = len(s.previewRequests) - 1
		}
		return
	}
	s.collectionReqIdx = -1
}

func (s *Shell) handleCollectionsKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "j", "down":
		s.collectionsCursorDown()
	case "k", "up":
		s.collectionsCursorUp()
	case "enter":
		if s.collectionReqIdx >= 0 && s.collectionReqIdx < len(s.previewRequests) {
			reqs := append([]httpfile.Request(nil), s.previewRequests...)
			s.loadRequestIntoEditor(s.currentCollectionName(), reqs, s.collectionReqIdx)
			s.setFocus(PanelRequest)
		}
	case "n":
		return s.newRequestInCurrentCollection()
	case "N":
		s.overlay = overlayNewCollection
		s.input = ""
	case "c":
		return s.duplicateCurrentPreviewRequest()
	case "d", "x":
		if s.collectionReqIdx >= 0 && s.collectionReqIdx < len(s.previewRequests) {
			s.overlay = overlayConfirmDelete
		}
	case "E":
		if s.currentCollectionName() != "" {
			s.overlay = overlayEnvSelect
		}
	}
	return nil
}

// newRequestInCurrentCollection creates an empty request in the currently
// expanded collection, loads it directly into the [0] Request panel, and
// moves focus there (it stays unsaved on disk until ctrl+s).
func (s *Shell) newRequestInCurrentCollection() tea.Cmd {
	name := s.currentCollectionName()
	if name == "" {
		return s.setStatus("先にコレクションを作成してください ('N')")
	}
	reqs := append([]httpfile.Request(nil), s.previewRequests...)
	reqs = append(reqs, httpfile.Request{Method: "GET"})
	s.loadRequestIntoEditor(name, reqs, len(reqs)-1)
	s.setFocus(PanelRequest)
	return nil
}

func (s *Shell) duplicateCurrentPreviewRequest() tea.Cmd {
	if s.collectionReqIdx < 0 || s.collectionReqIdx >= len(s.previewRequests) {
		return nil
	}
	if err := s.colStore.DuplicateRequest(s.currentCollectionName(), s.collectionReqIdx); err != nil {
		return s.setStatus(err.Error())
	}
	if err := s.reloadCollectionPreview(); err != nil {
		return s.setStatus(err.Error())
	}
	return s.setStatus("")
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
			s.savingViaNewCollection = false
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
					if s.savingViaNewCollection {
						cmd = s.finishScratchSave(name)
					} else {
						_ = s.reloadCollectionPreview()
					}
				}
			}
			s.savingViaNewCollection = false
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

	case overlaySaveTo:
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
				s.savingViaNewCollection = true
				s.overlay = overlayNewCollection
				s.input = ""
				return nil
			}
			if s.saveIdx < len(s.collections) {
				cmd := s.finishScratchSave(s.collections[s.saveIdx].Name)
				s.overlay = overlayNone
				return cmd
			}
			s.overlay = overlayNone
		}
		return nil

	case overlayRequestName:
		switch msg.String() {
		case "esc":
			s.overlay = overlayNone
		case "enter":
			name := strings.TrimSpace(s.input)
			if name == "" {
				return nil
			}
			if s.usingScratch {
				s.scratchRequest.Name = name
				s.editor.Name = name
				return s.beginScratchSave()
			}
			if s.requestIdx < len(s.requests) {
				s.requests[s.requestIdx].Name = name
			}
			s.editor.Name = name
			s.overlay = overlayNone
			return s.saveLoadedRequests()
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

	case overlayConfirmDelete:
		switch msg.String() {
		case "y":
			var cmd tea.Cmd
			name := s.currentCollectionName()
			if err := s.colStore.DeleteRequest(name, s.collectionReqIdx); err != nil {
				cmd = s.setStatus(err.Error())
			} else if err := s.reloadCollectionPreview(); err != nil {
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

// finishScratchSave appends the scratch request to collectionName's `.http`
// file, then focuses the Collections panel on that collection with the
// newly-saved request selected, and resets the scratch request to empty.
func (s *Shell) finishScratchSave(collectionName string) tea.Cmd {
	if err := s.colStore.CreateRequest(collectionName, s.scratchRequest); err != nil {
		return s.setStatus(err.Error())
	}
	for i, c := range s.collections {
		if c.Name == collectionName {
			s.collectionIdx = i
		}
	}
	if err := s.reloadCollectionPreview(); err != nil {
		return s.setStatus(err.Error())
	}
	s.collectionReqIdx = max0(len(s.previewRequests) - 1)
	s.scratchRequest = httpfile.Request{Method: "GET"}
	s.usingScratch = true
	s.setFocus(PanelCollections)
	return nil
}

func (s *Shell) cancel() {
	if s.cancelSend != nil {
		s.cancelSend()
	}
}

// sendLoadedCurrent expands the loaded collection request's variables
// against its active environment and, if fully defined, executes it via
// curl asynchronously.
func (s *Shell) sendLoadedCurrent() tea.Cmd {
	if s.sending || s.requestIdx >= len(s.requests) {
		return nil
	}
	collectionName := s.loadedCollection
	req := s.requests[s.requestIdx]

	vars := map[string]string{}
	if activeEnv, err := s.envStore.ActiveEnvironment(collectionName); err == nil && activeEnv != "" {
		v, err := s.envStore.Load(collectionName, activeEnv)
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

// sendScratchCurrent executes the scratch request as-is, with no
// {{variable}} expansion (it isn't tied to a collection or environment).
func (s *Shell) sendScratchCurrent() tea.Cmd {
	if s.sending {
		return nil
	}
	req := s.scratchRequest
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

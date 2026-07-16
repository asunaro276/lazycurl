package form

import (
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

var Methods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

// tab identifies which of the lower panels (Params/Headers/Auth/Body) is
// currently shown.
type tab int

const (
	TabParams tab = iota
	TabHeaders
	TabAuth
	TabBody
)

var tabLabels = []string{"Params", "Headers", "Auth", "Body"}

// focusZone identifies which top-level part of the form the normal-state
// cursor rests on.
type focusZone int

const (
	focusMethod focusZone = iota
	focusURL
	focusContent
)

var authTypes = []httpfile.AuthType{httpfile.AuthNone, httpfile.AuthBasic, httpfile.AuthBearer}

// editorDoneMsg is emitted internally after an external $EDITOR process
// exits, carrying the reloaded body content back into the model.
type editorDoneMsg struct {
	body string
	err  error
}

// Editor is the hybrid request-editing form: Method/URL/Params/Headers/Auth
// are edited via form controls, Body via a textarea with an external
// $EDITOR escape hatch (ctrl-e).
type Editor struct {
	// Name is the request's name. It is not part of the form's focus
	// cycle or rendered UI -- it is injected via FromRequest (and read
	// back via ToRequest) by callers that own the save-time name prompt
	// (see internal/tui/shell's overlayRequestName).
	Name       string
	methodIdx  int
	url        textinput.Model
	params     KVGrid
	headers    KVGrid
	authTypeI  int
	authUser   textinput.Model
	authPass   textinput.Model
	authToken  textinput.Model
	authField  int // 0=type selector, 1=first credential field, 2=second
	body       textarea.Model
	pragTO     textinput.Model
	pragK      bool
	pragNoRdir bool

	focus focusZone
	// editing is the form's insert state: false (normal) means movement
	// keys (j/k, h/l, [/]) navigate between Method/URL/tab content and
	// switch tabs without touching any value; true (insert) means the
	// field at e.focus owns keystrokes. This mirrors KVGrid's own
	// editing/non-editing split, raised one level to cover the whole form.
	editing bool
	tab     tab

	width, height int
	err           error
}

// New returns an empty Editor ready for a new request.
func New() Editor {
	u := textinput.New()
	u.Placeholder = "https://{{host}}/path"

	body := textarea.New()
	body.Placeholder = "(request body)"

	to := textinput.New()
	to.Placeholder = "e.g. 5s"

	user := textinput.New()
	user.Placeholder = "username"
	pass := textinput.New()
	pass.Placeholder = "password"
	pass.EchoMode = textinput.EchoPassword
	token := textinput.New()
	token.Placeholder = "token"

	return Editor{
		url:       u,
		params:    NewKVGrid(),
		headers:   NewKVGrid(),
		authUser:  user,
		authPass:  pass,
		authToken: token,
		body:      body,
		pragTO:    to,
		focus:     focusMethod,
		tab:       TabParams,
	}
}

// FromRequest loads req into the form.
func FromRequest(req httpfile.Request) Editor {
	e := New()
	e.Name = req.Name
	for i, m := range Methods {
		if m == req.Method {
			e.methodIdx = i
			break
		}
	}
	base, params := splitURL(req.URL)
	e.url.SetValue(base)
	e.params.Rows = params
	e.headers.Rows = append([]httpfile.KV(nil), req.Headers...)

	for i, t := range authTypes {
		if t == req.Auth.Type {
			e.authTypeI = i
			break
		}
	}
	e.authUser.SetValue(req.Auth.Username)
	e.authPass.SetValue(req.Auth.Password)
	e.authToken.SetValue(req.Auth.Token)

	e.body.SetValue(req.Body)
	e.pragK = req.Pragmas.Insecure
	e.pragTO.SetValue(req.Pragmas.Timeout)
	e.pragNoRdir = req.Pragmas.NoRedirect
	return e
}

// ToRequest converts the current form state back into a Request, ready for
// serialization or execution.
func (e Editor) ToRequest() httpfile.Request {
	return httpfile.Request{
		Name:    e.Name,
		Method:  Methods[e.methodIdx],
		URL:     joinURL(e.url.Value(), e.params.Rows),
		Headers: append([]httpfile.KV(nil), e.headers.Rows...),
		Auth: httpfile.Auth{
			Type:     authTypes[e.authTypeI],
			Username: e.authUser.Value(),
			Password: e.authPass.Value(),
			Token:    e.authToken.Value(),
		},
		Body: e.body.Value(),
		Pragmas: httpfile.Pragmas{
			Insecure:   e.pragK,
			Timeout:    e.pragTO.Value(),
			NoRedirect: e.pragNoRdir,
		},
	}
}

func (e *Editor) SetSize(w, h int) {
	e.width, e.height = w, h
	e.url.Width = max(10, w-10)
	e.body.SetWidth(w)
	e.body.SetHeight(max(3, h-10))
}

// FocusNext moves the normal-state cursor forward: Method -> URL -> content
// -> Method. It never touches insert/blur state -- fields only receive
// keyboard input once enterInsert focuses them explicitly.
func (e *Editor) FocusNext() {
	switch e.focus {
	case focusMethod:
		e.focus = focusURL
	case focusURL:
		e.focus = focusContent
	case focusContent:
		e.focus = focusMethod
	}
}

// FocusPrev moves the normal-state cursor backward, the reverse of
// FocusNext.
func (e *Editor) FocusPrev() {
	switch e.focus {
	case focusMethod:
		e.focus = focusContent
	case focusURL:
		e.focus = focusMethod
	case focusContent:
		e.focus = focusURL
	}
}

func (e *Editor) focusActiveTab() {
	switch e.tab {
	case TabParams:
		e.params.Focus()
	case TabHeaders:
		e.headers.Focus()
	case TabAuth:
		e.focusAuthField()
	case TabBody:
		e.body.Focus()
	}
}

func (e *Editor) focusAuthField() {
	e.authUser.Blur()
	e.authPass.Blur()
	e.authToken.Blur()
	if authTypes[e.authTypeI] == httpfile.AuthBasic {
		if e.authField == 1 {
			e.authUser.Focus()
		} else if e.authField == 2 {
			e.authPass.Focus()
		}
	} else if authTypes[e.authTypeI] == httpfile.AuthBearer {
		if e.authField == 1 {
			e.authToken.Focus()
		}
	}
}

// SetTab switches the active lower panel and resets its Auth sub-state.
// Insert focus, if any, is left to the caller (enterInsert/updateEditing) --
// SetTab itself never grabs or releases keyboard input.
func (e *Editor) SetTab(t tab) {
	e.tab = t
	e.authField = 0
}

// Editing reports whether some part of the form -- a text field, the
// textarea, an Auth credential, or a KVGrid cell -- currently owns
// keystrokes (the form's insert state). Panel-level shortcuts must not fire
// while this is true.
func (e Editor) Editing() bool { return e.editing }

// enterInsert begins the insert state for whichever field the normal-state
// cursor currently rests on. Method has no insert state of its own -- its
// value changes directly via h/l while normal, mirroring KVGrid's enabled
// checkbox column -- so entering insert there is a no-op.
func (e *Editor) enterInsert() {
	switch e.focus {
	case focusURL:
		e.editing = true
		e.url.Focus()
	case focusContent:
		e.editing = true
		e.focusActiveTab()
	}
}

// Update handles a bubbletea message, dispatching to the form's normal or
// insert state.
func (e Editor) Update(msg tea.Msg) (Editor, tea.Cmd) {
	switch msg := msg.(type) {
	case editorDoneMsg:
		if msg.err == nil {
			e.body.SetValue(msg.body)
		} else {
			e.err = msg.err
		}
		return e, nil

	case tea.KeyMsg:
		if e.editing {
			return e.updateEditing(msg)
		}
		return e.updateNormal(msg)
	}
	return e, nil
}

// updateNormal handles the form's non-insert state: movement between
// Method/URL/content, Method's direct h/l value cycling, tab switching, and
// enter to begin insert on the focused field.
func (e Editor) updateNormal(msg tea.KeyMsg) (Editor, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		e.FocusNext()
	case "k", "up":
		e.FocusPrev()
	case "h", "left":
		if e.focus == focusMethod {
			e.methodIdx = (e.methodIdx - 1 + len(Methods)) % len(Methods)
		}
	case "l", "right":
		if e.focus == focusMethod {
			e.methodIdx = (e.methodIdx + 1) % len(Methods)
		}
	case "[", "]":
		n := tab(len(tabLabels))
		if msg.String() == "]" {
			e.SetTab((e.tab + 1) % n)
		} else {
			e.SetTab((e.tab - 1 + n) % n)
		}
	case "enter":
		e.enterInsert()
	}
	return e, nil
}

// updateEditing handles the form's insert state: keystrokes belong to
// whichever field is focused, except esc, which pops exactly one level
// (KVGrid cell-edit -> KVGrid row nav -> form normal; Auth credential ->
// Auth type selector -> form normal) per field.
func (e Editor) updateEditing(msg tea.KeyMsg) (Editor, tea.Cmd) {
	if msg.String() == "ctrl+e" && e.focus == focusContent && e.tab == TabBody {
		return e, e.openExternalEditor()
	}

	switch e.focus {
	case focusURL:
		if msg.String() == "esc" {
			e.editing = false
			e.url.Blur()
			return e, nil
		}
		var cmd tea.Cmd
		e.url, cmd = e.url.Update(msg)
		return e, cmd

	case focusContent:
		return e.updateContentEditing(msg)
	}
	return e, nil
}

func (e Editor) updateContentEditing(msg tea.KeyMsg) (Editor, tea.Cmd) {
	switch e.tab {
	case TabParams:
		if msg.String() == "esc" && !e.params.Editing() {
			e.editing = false
			e.params.Blur()
			return e, nil
		}
		var cmd tea.Cmd
		e.params, cmd = e.params.Update(msg)
		return e, cmd

	case TabHeaders:
		if msg.String() == "esc" && !e.headers.Editing() {
			e.editing = false
			e.headers.Blur()
			return e, nil
		}
		var cmd tea.Cmd
		e.headers, cmd = e.headers.Update(msg)
		return e, cmd

	case TabAuth:
		if msg.String() == "esc" && e.authField == 0 {
			e.editing = false
			return e, nil
		}
		return e.updateAuth(msg)

	case TabBody:
		if msg.String() == "esc" {
			e.editing = false
			e.body.Blur()
			return e, nil
		}
		var cmd tea.Cmd
		e.body, cmd = e.body.Update(msg)
		return e, cmd
	}
	return e, nil
}

func (e Editor) updateAuth(msg tea.Msg) (Editor, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		if e.authField > 0 {
			var cmd tea.Cmd
			if authTypes[e.authTypeI] == httpfile.AuthBasic {
				if e.authField == 1 {
					e.authUser, cmd = e.authUser.Update(msg)
				} else {
					e.authPass, cmd = e.authPass.Update(msg)
				}
			} else {
				e.authToken, cmd = e.authToken.Update(msg)
			}
			return e, cmd
		}
		return e, nil
	}

	if e.authField == 0 {
		switch km.String() {
		case "left", "h":
			e.authTypeI = (e.authTypeI - 1 + len(authTypes)) % len(authTypes)
			return e, nil
		case "right", "l":
			e.authTypeI = (e.authTypeI + 1) % len(authTypes)
			return e, nil
		case "down", "j", "enter":
			if authTypes[e.authTypeI] != httpfile.AuthNone {
				e.authField = 1
				e.focusAuthField()
			}
			return e, nil
		}
		return e, nil
	}

	switch km.String() {
	case "esc", "up":
		if km.String() == "up" && e.authField > 1 {
			e.authField--
			e.focusAuthField()
			return e, nil
		}
		e.authField = 0
		e.authUser.Blur()
		e.authPass.Blur()
		e.authToken.Blur()
		return e, nil
	case "down":
		if authTypes[e.authTypeI] == httpfile.AuthBasic && e.authField < 2 {
			e.authField++
			e.focusAuthField()
		}
		return e, nil
	}

	var cmd tea.Cmd
	if authTypes[e.authTypeI] == httpfile.AuthBasic {
		if e.authField == 1 {
			e.authUser, cmd = e.authUser.Update(msg)
		} else {
			e.authPass, cmd = e.authPass.Update(msg)
		}
	} else {
		e.authToken, cmd = e.authToken.Update(msg)
	}
	return e, cmd
}

// openExternalEditor writes the current body to a temp file, launches
// $EDITOR on it as a foreground process, and reloads the content once the
// editor exits.
func (e *Editor) openExternalEditor() tea.Cmd {
	editorBin := os.Getenv("EDITOR")
	if editorBin == "" {
		editorBin = "vi"
	}

	tmp, err := os.CreateTemp("", "lazycurl-body-*.txt")
	if err != nil {
		return func() tea.Msg { return editorDoneMsg{err: err} }
	}
	if _, err := tmp.WriteString(e.body.Value()); err != nil {
		tmp.Close()
		return func() tea.Msg { return editorDoneMsg{err: err} }
	}
	path := tmp.Name()
	tmp.Close()

	cmd := exec.Command(editorBin, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(path)
		if err != nil {
			return editorDoneMsg{err: err}
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return editorDoneMsg{err: readErr}
		}
		return editorDoneMsg{body: string(data)}
	})
}

var (
	styleTabActive   = lipgloss.NewStyle().Bold(true).Underline(true)
	styleTabInactive = lipgloss.NewStyle().Faint(true)
	styleFieldLabel  = lipgloss.NewStyle().Bold(true)
	styleFocusBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	stylePlainBorder = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
)

// View renders the form.
func (e Editor) View() string {
	var b strings.Builder

	methodView := Methods[e.methodIdx]
	if e.focus == focusMethod {
		methodView = styleFocusBorder.Render(methodView)
	} else {
		methodView = stylePlainBorder.Render(methodView)
	}

	urlView := e.url.View()
	if e.focus == focusURL {
		urlView = styleFocusBorder.Render(urlView)
	} else {
		urlView = stylePlainBorder.Render(urlView)
	}

	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, methodView, " ", urlView))
	b.WriteString("\n\n")

	var tabsRendered []string
	for i, label := range tabLabels {
		if tab(i) == e.tab {
			tabsRendered = append(tabsRendered, styleTabActive.Render(label))
		} else {
			tabsRendered = append(tabsRendered, styleTabInactive.Render(label))
		}
	}
	b.WriteString(strings.Join(tabsRendered, "  "))
	b.WriteString("\n\n")

	switch e.tab {
	case TabParams:
		b.WriteString(e.params.View("Param", "Value"))
	case TabHeaders:
		b.WriteString(e.headers.View("Key", "Value"))
	case TabAuth:
		b.WriteString(e.viewAuth())
	case TabBody:
		b.WriteString(e.body.View())
	}

	return b.String()
}

func (e Editor) viewAuth() string {
	var b strings.Builder
	typeLabel := "Type: " + string(authTypes[e.authTypeI])
	if e.authField == 0 && e.focus == focusContent {
		typeLabel = styleSelected.Render(typeLabel)
	}
	b.WriteString(typeLabel)
	b.WriteString("\n\n")

	switch authTypes[e.authTypeI] {
	case httpfile.AuthBasic:
		b.WriteString(styleFieldLabel.Render("Username: ") + e.authUser.View())
		b.WriteString("\n")
		b.WriteString(styleFieldLabel.Render("Password: ") + e.authPass.View())
	case httpfile.AuthBearer:
		b.WriteString(styleFieldLabel.Render("Token: ") + e.authToken.View())
	}
	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

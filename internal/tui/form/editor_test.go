package form

import (
	"strings"
	"testing"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// TestEditorStartsInNormalState confirms a fresh Editor starts on Method,
// not editing (insert).
func TestEditorStartsInNormalState(t *testing.T) {
	e := New()
	if e.focus != focusMethod {
		t.Fatalf("expected fresh Editor to start focused on Method, got %v", e.focus)
	}
	if e.Editing() {
		t.Fatal("expected fresh Editor to start in normal state, not insert")
	}
}

// TestEditorNormalNavigationCyclesFields confirms j/k cycle Method -> URL ->
// content -> Method (and back) without entering insert.
func TestEditorNormalNavigationCyclesFields(t *testing.T) {
	e := New()

	e, _ = e.Update(key("j"))
	if e.focus != focusURL || e.Editing() {
		t.Fatalf("expected URL focus after j, still normal; got focus=%v editing=%v", e.focus, e.Editing())
	}

	e, _ = e.Update(key("j"))
	if e.focus != focusContent || e.Editing() {
		t.Fatalf("expected content focus after second j, still normal; got focus=%v editing=%v", e.focus, e.Editing())
	}

	e, _ = e.Update(key("j"))
	if e.focus != focusMethod {
		t.Fatalf("expected j to wrap back to Method, got %v", e.focus)
	}

	e, _ = e.Update(key("k"))
	if e.focus != focusContent {
		t.Fatalf("expected k from Method to wrap backward to content, got %v", e.focus)
	}
}

// TestEditorMethodChangesDirectlyInNormalState confirms h/l cycle the
// Method value while normal, with no insert step required -- mirroring
// KVGrid's enabled-checkbox column, which toggles without entering edit.
func TestEditorMethodChangesDirectlyInNormalState(t *testing.T) {
	e := New() // starts at Methods[0] == "GET", focus == Method
	e, _ = e.Update(key("l"))
	if got := Methods[e.methodIdx]; got != "POST" {
		t.Fatalf("expected l to advance Method to POST, got %q", got)
	}
	if e.Editing() {
		t.Fatal("expected Method h/l to work without entering insert")
	}
	e, _ = e.Update(key("h"))
	if got := Methods[e.methodIdx]; got != "GET" {
		t.Fatalf("expected h to move Method back to GET, got %q", got)
	}
}

// TestEditorURLInsertRoundTrip confirms enter starts insert on URL, typed
// characters are captured, and esc returns to normal.
func TestEditorURLInsertRoundTrip(t *testing.T) {
	e := New()
	e, _ = e.Update(key("j")) // Method -> URL

	e, _ = e.Update(key("enter"))
	if !e.Editing() {
		t.Fatal("expected enter on URL to start insert")
	}

	for _, r := range "https://x.test" {
		e, _ = e.Update(key(string(r)))
	}
	if e.url.Value() != "https://x.test" {
		t.Fatalf("expected typed text in URL, got %q", e.url.Value())
	}

	e, _ = e.Update(key("esc"))
	if e.Editing() {
		t.Fatal("expected esc to return URL to normal state")
	}
	if e.url.Value() != "https://x.test" {
		t.Fatalf("expected URL value to survive leaving insert, got %q", e.url.Value())
	}
}

// TestEditorParamsGridNestsTwoLevelsDeep exercises the Params tab's nested
// insert: Editor normal -> enter -> KVGrid focused (row/col nav, still
// Editor-insert) -> enter -> KVGrid cell edit (deeper nest). Each esc must
// pop exactly one level, per design's "esc pops one level" rule.
func TestEditorParamsGridNestsTwoLevelsDeep(t *testing.T) {
	e := New()
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Params by default)

	e, _ = e.Update(key("enter")) // enter the grid: row/col nav level
	if !e.Editing() {
		t.Fatal("expected entering the Params tab to start Editor insert")
	}
	if !e.params.Focused() {
		t.Fatal("expected the grid to be focused once inside")
	}
	if e.params.Editing() {
		t.Fatal("expected the grid to start at row/col nav, not mid-cell-edit")
	}

	e, _ = e.Update(key("a")) // add a row -> grid begins editing its key cell
	if !e.params.Editing() {
		t.Fatal("expected 'a' to start editing the new row's key (nested one level deeper)")
	}
	if !e.Editing() {
		t.Fatal("expected Editor.Editing() to stay true while the grid cell is mid-edit")
	}

	e, _ = e.Update(key("X"))
	e, _ = e.Update(key("esc")) // first esc: cancel the cell edit, stay in grid nav
	if e.params.Editing() {
		t.Fatal("expected esc to cancel the cell edit (one level), not exit the grid")
	}
	if !e.Editing() {
		t.Fatal("expected Editor.Editing() to remain true after popping only the cell-edit level")
	}

	e, _ = e.Update(key("esc")) // second esc: exit the grid back to Editor normal
	if e.Editing() {
		t.Fatal("expected the second esc to pop out of the grid to Editor normal state")
	}
	if e.params.Focused() {
		t.Fatal("expected the grid to be blurred after exiting to Editor normal")
	}
}

// TestEditorAuthNestsSelectorAndCredential mirrors the KVGrid nesting test
// for the Auth tab: Editor normal -> enter -> type selector (still
// Editor-insert) -> enter -> credential field (deeper nest); each esc pops
// one level.
func TestEditorAuthNestsSelectorAndCredential(t *testing.T) {
	e := New()
	e.SetTab(TabAuth)
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Auth)

	e, _ = e.Update(key("enter")) // enter Auth content: type-selector level
	if !e.Editing() {
		t.Fatal("expected entering the Auth tab to start Editor insert")
	}
	if e.authField != 0 {
		t.Fatalf("expected authField to start at the type selector (0), got %d", e.authField)
	}

	e, _ = e.Update(key("l")) // AuthNone -> AuthBasic
	if authTypes[e.authTypeI] != httpfile.AuthBasic {
		t.Fatalf("expected l to advance the auth type selector to Basic, got %v", authTypes[e.authTypeI])
	}

	e, _ = e.Update(key("enter")) // selector -> first credential field (deeper nest)
	if e.authField != 1 {
		t.Fatalf("expected enter to move into the first credential field, got authField=%d", e.authField)
	}
	if !e.Editing() {
		t.Fatal("expected Editor.Editing() to stay true while inside a credential field")
	}

	e, _ = e.Update(key("esc")) // first esc: credential field -> selector (one level)
	if e.authField != 0 {
		t.Fatalf("expected esc to pop back to the type selector, got authField=%d", e.authField)
	}
	if !e.Editing() {
		t.Fatal("expected Editor.Editing() to remain true after popping only the credential level")
	}

	e, _ = e.Update(key("esc")) // second esc: selector -> Editor normal
	if e.Editing() {
		t.Fatal("expected the second esc to exit Auth content to Editor normal state")
	}
}

// TestEditorOptionsTogglesPragmas exercises the Options tab: enter into
// Level 1 (row nav), toggling Stream/Insecure/No-redirect, and editing
// Timeout as a nested Level 2 text field.
func TestEditorOptionsTogglesPragmas(t *testing.T) {
	e := New()
	e.SetTab(TabOptions)
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Options)

	e, _ = e.Update(key("enter")) // enter Options: row nav level
	if !e.Editing() {
		t.Fatal("expected entering the Options tab to start Editor insert")
	}

	e, _ = e.Update(key("enter")) // toggle Stream (row 0)
	if !e.pragStream {
		t.Fatal("expected enter on the Stream row to enable it")
	}

	e, _ = e.Update(key("j")) // Stream -> Insecure
	e, _ = e.Update(key(" ")) // toggle Insecure via space
	if !e.pragK {
		t.Fatal("expected space on the Insecure row to enable it")
	}

	e, _ = e.Update(key("j"))     // Insecure -> No-redirect
	e, _ = e.Update(key("j"))     // No-redirect -> Timeout
	e, _ = e.Update(key("enter")) // start editing Timeout (Level 2)
	if !e.optionsEditingTO {
		t.Fatal("expected enter on the Timeout row to start text editing")
	}

	for _, r := range "5s" {
		e, _ = e.Update(key(string(r)))
	}
	e, _ = e.Update(key("enter")) // commit Timeout
	if e.optionsEditingTO {
		t.Fatal("expected enter to commit Timeout and leave text editing")
	}
	if e.pragTO.Value() != "5s" {
		t.Fatalf("expected Timeout value %q, got %q", "5s", e.pragTO.Value())
	}
	if !e.Editing() {
		t.Fatal("expected Editor.Editing() to remain true after leaving Timeout's text edit")
	}

	e, _ = e.Update(key("esc")) // exit Options back to Editor normal
	if e.Editing() {
		t.Fatal("expected esc to pop out of Options to Editor normal state")
	}

	got := e.ToRequest().Pragmas
	if !got.Stream || !got.Insecure || got.Timeout != "5s" {
		t.Fatalf("expected Options edits reflected in ToRequest, got %+v", got)
	}
}

// TestEditorBodyInsertRoundTrip confirms the Body tab's single-level insert:
// enter starts it, esc ends it, with no intermediate nesting.
func TestEditorBodyInsertRoundTrip(t *testing.T) {
	e := New()
	e.SetTab(TabBody)
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Body)

	e, _ = e.Update(key("enter"))
	if !e.Editing() {
		t.Fatal("expected enter on Body to start insert")
	}

	e, _ = e.Update(key("esc"))
	if e.Editing() {
		t.Fatal("expected esc to end Body insert")
	}
}

// TestEditorPragmaRoundTrip confirms all four pragmas, including Stream,
// survive a FromRequest -> (navigation) -> ToRequest round trip. Stream in
// particular used to be dropped by ToRequest, silently downgrading a
// `@stream` request to a batch send the moment the form was touched.
func TestEditorPragmaRoundTrip(t *testing.T) {
	e := FromRequest(httpfile.Request{
		Method: "GET",
		URL:    "https://example.com",
		Pragmas: httpfile.Pragmas{
			Stream:     true,
			Insecure:   true,
			Timeout:    "5s",
			NoRedirect: true,
		},
	})

	// Merely navigating the form (no edits) must not lose any pragma.
	e, _ = e.Update(key("j"))
	e, _ = e.Update(key("j"))
	e, _ = e.Update(key("k"))

	got := e.ToRequest().Pragmas
	want := httpfile.Pragmas{Stream: true, Insecure: true, Timeout: "5s", NoRedirect: true}
	if got != want {
		t.Fatalf("expected pragmas to survive navigation unchanged, got %+v, want %+v", got, want)
	}
}

// TestEditorTabSwitchOnlyInNormalState confirms h/l switch tabs while
// Level 0 focus rests on content, and are instead typed literally once
// insert has captured a text field (here, URL).
func TestEditorTabSwitchOnlyInNormalState(t *testing.T) {
	e := New()
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Params)

	e, _ = e.Update(key("l"))
	if e.tab != TabHeaders {
		t.Fatalf("expected l to switch from Params to Headers in normal state, got %v", e.tab)
	}
	e, _ = e.Update(key("h"))
	if e.tab != TabParams {
		t.Fatalf("expected h to switch back from Headers to Params, got %v", e.tab)
	}

	e, _ = e.Update(key("k"))     // content -> URL (back out, away from h/l = tab switch)
	e, _ = e.Update(key("enter")) // start insert on URL
	e, _ = e.Update(key("l"))
	if e.url.Value() != "l" {
		t.Fatalf("expected l typed literally into URL while inserting, got %q", e.url.Value())
	}
	if e.tab != TabParams {
		t.Fatalf("expected tab switch to be inert while inserting text, got %v", e.tab)
	}
}

// TestEditorTabSwitchWrapsThroughAllFiveTabs confirms the tab cycle now
// includes Options and wraps in both directions.
func TestEditorTabSwitchWrapsThroughAllFiveTabs(t *testing.T) {
	e := New()
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Params)

	order := []tab{TabHeaders, TabAuth, TabBody, TabOptions, TabParams}
	for _, want := range order {
		e, _ = e.Update(key("l"))
		if e.tab != want {
			t.Fatalf("expected l to advance to %v, got %v", want, e.tab)
		}
	}

	e, _ = e.Update(key("h"))
	if e.tab != TabOptions {
		t.Fatalf("expected h from Params to wrap backward to Options, got %v", e.tab)
	}
}

// TestEditorMethodShowsArrowsWhenFocused confirms the Method field renders
// with left/right arrows only while it holds Level 0 focus, as a visual
// hint that h/l changes its value.
func TestEditorMethodShowsArrowsWhenFocused(t *testing.T) {
	e := New()
	if !strings.Contains(e.View(), "◀") || !strings.Contains(e.View(), "▶") {
		t.Fatalf("expected Method arrows while focused, got %q", e.View())
	}

	e, _ = e.Update(key("j")) // Method -> URL
	if strings.Contains(e.View(), "◀") {
		t.Fatalf("expected no Method arrows once focus leaves Method, got %q", e.View())
	}
}

// TestEditorGridColumnMoveDoesNotLeakIntoTabSwitch confirms that once Level 1
// (KVGrid row/col nav) owns h/l, tab switching only resumes after esc pops
// back out to Level 0 -- the two never fire on the same keystroke.
func TestEditorGridColumnMoveDoesNotLeakIntoTabSwitch(t *testing.T) {
	e := New()
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Params)
	e.params.Rows = []httpfile.KV{{Enabled: true, Key: "X", Value: "Y"}}

	e, _ = e.Update(key("enter")) // Level 0 -> Level 1 (grid row/col nav)
	e, _ = e.Update(key("l"))     // column move within the grid, not a tab switch
	if e.tab != TabParams {
		t.Fatalf("expected l inside the grid to move columns, not switch tabs; got tab=%v", e.tab)
	}
	if e.params.cursorCol != colValue {
		t.Fatalf("expected l inside the grid to move the cursor to colValue, got %v", e.params.cursorCol)
	}

	e, _ = e.Update(key("esc")) // Level 1 -> Level 0
	e, _ = e.Update(key("l"))   // now h/l means tab switch again
	if e.tab != TabHeaders {
		t.Fatalf("expected l after esc to switch tabs, got %v", e.tab)
	}
}

// TestEditorAuthTypeSelectDoesNotLeakIntoTabSwitch mirrors the KVGrid test
// for Auth's type selector, which also owns h/l once inside Level 1.
func TestEditorAuthTypeSelectDoesNotLeakIntoTabSwitch(t *testing.T) {
	e := New()
	e.SetTab(TabAuth)
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Auth)

	e, _ = e.Update(key("enter")) // Level 0 -> Level 1 (type selector)
	e, _ = e.Update(key("l"))     // cycles the auth type, not a tab switch
	if e.tab != TabAuth {
		t.Fatalf("expected l inside Auth's type selector to not switch tabs, got %v", e.tab)
	}
	if authTypes[e.authTypeI] != httpfile.AuthBasic {
		t.Fatalf("expected l to advance the auth type, got %v", authTypes[e.authTypeI])
	}

	e, _ = e.Update(key("esc")) // Level 1 -> Level 0
	e, _ = e.Update(key("l"))   // now h/l means tab switch again
	if e.tab != TabBody {
		t.Fatalf("expected l after esc to switch tabs, got %v", e.tab)
	}
}

// TestEditorLevel1HighlightsImmediatelyOnEntry confirms Params/Headers and
// Auth show their cursor/selector highlighted as soon as Level 1 is
// entered, with no further keystroke required to reveal where the cursor
// is.
func TestEditorLevel1HighlightsImmediatelyOnEntry(t *testing.T) {
	e := New()
	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Params)
	e.params.Rows = []httpfile.KV{{Enabled: true, Key: "X-Key", Value: "some-value"}}

	e, _ = e.Update(key("enter")) // Level 0 -> Level 1
	if !strings.Contains(e.View(), renderKVCell("X-Key", true, false)) {
		t.Fatalf("expected the Key cell highlighted immediately on entering the grid, got %q", e.View())
	}

	e.SetTab(TabAuth)
	e, _ = e.Update(key("enter")) // Level 0 -> Level 1 (type selector)
	if !strings.Contains(e.View(), styleSelected.Render("Type: "+string(httpfile.AuthNone))) {
		t.Fatalf("expected the Auth type selector highlighted immediately on entering, got %q", e.View())
	}
}

// TestEditorFooterHintReflectsLevelAndTab confirms FooterHint changes with
// the form's Level 0/1/2 state and active tab, rather than only the
// top-level Editing() bool.
func TestEditorFooterHintReflectsLevelAndTab(t *testing.T) {
	e := New()
	if hint := e.FooterHint(); !strings.Contains(hint, "Method変更") {
		t.Fatalf("expected Method-focused hint to mention Method変更, got %q", hint)
	}

	e, _ = e.Update(key("j")) // Method -> URL
	e, _ = e.Update(key("j")) // URL -> content (Params)
	if hint := e.FooterHint(); !strings.Contains(hint, "タブ切替") {
		t.Fatalf("expected content-focused hint to mention タブ切替, got %q", hint)
	}

	e, _ = e.Update(key("enter")) // Level 0 -> Level 1 (grid nav)
	hint := e.FooterHint()
	for _, want := range []string{"行移動", "列移動", "a", "d/x", "space"} {
		if !strings.Contains(hint, want) {
			t.Fatalf("expected grid-nav hint to mention %q, got %q", want, hint)
		}
	}

	e, _ = e.Update(key("a")) // start editing a cell (Level 2)
	if hint := e.FooterHint(); !strings.Contains(hint, "確定") {
		t.Fatalf("expected cell-edit hint to mention 確定, got %q", hint)
	}

	e2 := New()
	e2.SetTab(TabAuth)
	e2, _ = e2.Update(key("j"))     // Method -> URL
	e2, _ = e2.Update(key("j"))     // URL -> content (Auth)
	e2, _ = e2.Update(key("enter")) // Level 1: type selector
	if hint := e2.FooterHint(); !strings.Contains(hint, "タイプ選択") {
		t.Fatalf("expected Auth type-selector hint to mention タイプ選択, got %q", hint)
	}
}

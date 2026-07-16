package form

import (
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

// TestEditorTabSwitchOnlyInNormalState confirms [/] switch tabs while
// normal, and are instead typed literally once insert has captured a text
// field (here, URL).
func TestEditorTabSwitchOnlyInNormalState(t *testing.T) {
	e := New()
	e, _ = e.Update(key("]"))
	if e.tab != TabHeaders {
		t.Fatalf("expected ] to switch from Params to Headers in normal state, got %v", e.tab)
	}

	e, _ = e.Update(key("j"))     // Method -> URL
	e, _ = e.Update(key("enter")) // start insert on URL
	e, _ = e.Update(key("]"))
	if e.url.Value() != "]" {
		t.Fatalf("expected ] typed literally into URL while inserting, got %q", e.url.Value())
	}
	if e.tab != TabHeaders {
		t.Fatalf("expected tab switch to be inert while inserting text, got %v", e.tab)
	}
}

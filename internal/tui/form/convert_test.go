package form

import (
	"reflect"
	"testing"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

func TestSplitJoinURL(t *testing.T) {
	base, params := splitURL("https://api.example.com/users?active=true&role=admin")
	if base != "https://api.example.com/users" {
		t.Errorf("unexpected base: %q", base)
	}
	want := []httpfile.KV{
		{Key: "active", Value: "true", Enabled: true},
		{Key: "role", Value: "admin", Enabled: true},
	}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("unexpected params: %+v", params)
	}

	joined := joinURL(base, params)
	if joined != "https://api.example.com/users?active=true&role=admin" {
		t.Errorf("unexpected joined URL: %q", joined)
	}
}

func TestSplitURLNoQuery(t *testing.T) {
	base, params := splitURL("{{host}}/health")
	if base != "{{host}}/health" || params != nil {
		t.Errorf("unexpected: base=%q params=%+v", base, params)
	}
}

func TestJoinURLSkipsDisabled(t *testing.T) {
	params := []httpfile.KV{
		{Key: "a", Value: "1", Enabled: true},
		{Key: "b", Value: "2", Enabled: false},
	}
	got := joinURL("https://example.com", params)
	if got != "https://example.com?a=1" {
		t.Errorf("unexpected: %q", got)
	}
}

func TestEditorFromRequestToRequestRoundTrip(t *testing.T) {
	req := httpfile.Request{
		Name:   "Get user",
		Method: "GET",
		URL:    "{{host}}/users?id=42",
		Headers: []httpfile.KV{
			{Key: "X-Trace", Value: "abc", Enabled: true},
		},
		Auth: httpfile.Auth{Type: httpfile.AuthBearer, Token: "tok"},
		Body: "",
		Pragmas: httpfile.Pragmas{
			Insecure: true,
			Timeout:  "5s",
		},
	}

	e := FromRequest(req)
	got := e.ToRequest()

	if !reflect.DeepEqual(got, req) {
		t.Errorf("round trip mismatch:\n got: %+v\nwant: %+v", got, req)
	}
}

// TestEditorFocusCycleExcludesName confirms Name has no zone of its own in
// the form's focus cycle: a fresh Editor starts at Method (not a Name
// field), and cycling forward visits exactly Method -> URL -> content before
// wrapping back to Method.
func TestEditorFocusCycleExcludesName(t *testing.T) {
	e := New()
	if e.focus != focusMethod {
		t.Fatal("expected a fresh Editor to start at its first focus zone (Method)")
	}

	e.FocusNext() // Method -> URL
	if e.focus != focusURL {
		t.Fatal("expected URL to follow Method")
	}

	e.FocusNext() // URL -> content
	if e.focus != focusContent {
		t.Fatal("expected content to follow URL")
	}

	e.FocusNext() // content -> Method (wraps directly, no Name zone in between)
	if e.focus != focusMethod {
		t.Fatal("expected focus to cycle back to Method, skipping any Name zone")
	}
}

// TestEditorNameSurvivesFocusCycling confirms Name is carried through as an
// externally-injected value: it round-trips via FromRequest/ToRequest
// unchanged regardless of how the form's own focus/content fields are
// exercised.
func TestEditorNameSurvivesFocusCycling(t *testing.T) {
	e := FromRequest(httpfile.Request{Name: "Get user", Method: "GET"})
	e.FocusNext()
	e.FocusNext()
	e.FocusNext()

	if got := e.ToRequest().Name; got != "Get user" {
		t.Fatalf("expected Name to survive focus cycling untouched, got %q", got)
	}
}

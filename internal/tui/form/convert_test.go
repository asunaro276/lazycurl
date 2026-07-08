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

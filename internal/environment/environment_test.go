package environment

import (
	"path/filepath"
	"testing"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

func TestStoreListLoadSaveActive(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))

	if err := s.Save("api", "dev", map[string]string{"host": "https://dev.example.com"}); err != nil {
		t.Fatalf("Save dev: %v", err)
	}
	if err := s.Save("api", "prod", map[string]string{"host": "https://api.example.com"}); err != nil {
		t.Fatalf("Save prod: %v", err)
	}

	names, err := s.List("api")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 || names[0] != "dev" || names[1] != "prod" {
		t.Fatalf("unexpected names: %v", names)
	}

	active, err := s.ActiveEnvironment("api")
	if err != nil {
		t.Fatalf("ActiveEnvironment: %v", err)
	}
	if active != "" {
		t.Fatalf("expected no active environment yet, got %q", active)
	}

	if err := s.SetActiveEnvironment("api", "prod"); err != nil {
		t.Fatalf("SetActiveEnvironment: %v", err)
	}
	active, err = s.ActiveEnvironment("api")
	if err != nil {
		t.Fatalf("ActiveEnvironment: %v", err)
	}
	if active != "prod" {
		t.Fatalf("expected prod, got %q", active)
	}

	vars, err := s.Load("api", "prod")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if vars["host"] != "https://api.example.com" {
		t.Fatalf("unexpected vars: %v", vars)
	}
}

func TestExpand(t *testing.T) {
	vars := map[string]string{"host": "https://api.example.com", "id": "42"}

	result, undefined := Expand("{{host}}/users/{{id}}", vars)
	if result != "https://api.example.com/users/42" {
		t.Errorf("unexpected result: %q", result)
	}
	if len(undefined) != 0 {
		t.Errorf("expected no undefined vars, got %v", undefined)
	}

	result, undefined = Expand("{{host}}/users/{{missing}}", vars)
	if result != "https://api.example.com/users/{{missing}}" {
		t.Errorf("unexpected result: %q", result)
	}
	if len(undefined) != 1 || undefined[0] != "missing" {
		t.Errorf("expected [missing], got %v", undefined)
	}
}

func TestExpandRequest(t *testing.T) {
	req := httpfile.Request{
		URL: "{{host}}/users/{{id}}",
		Headers: []httpfile.KV{
			{Key: "X-Trace", Value: "{{trace}}", Enabled: true},
		},
		Body: `{"id": "{{id}}"}`,
	}
	vars := map[string]string{"host": "https://api.example.com", "id": "42"}

	out, undefined := ExpandRequest(req, vars)
	if out.URL != "https://api.example.com/users/42" {
		t.Errorf("unexpected URL: %q", out.URL)
	}
	if len(undefined) != 1 || undefined[0] != "trace" {
		t.Errorf("expected [trace] undefined, got %v", undefined)
	}
}

package collection

import (
	"path/filepath"
	"testing"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

func TestStoreListAndCRUD(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)

	if err := s.CreateCollection("api"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := s.CreateRequest("api", httpfile.Request{Name: "Get user", Method: "GET", URL: "https://example.com/users/1"}); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}
	if err := s.CreateRequest("api", httpfile.Request{Name: "Create user", Method: "POST", URL: "https://example.com/users"}); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}

	cols, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(cols) != 1 || cols[0].Name != "api" || cols[0].Path != filepath.Join(dir, "api.http") {
		t.Fatalf("unexpected collections: %+v", cols)
	}

	reqs, err := s.LoadRequests("api")
	if err != nil {
		t.Fatalf("LoadRequests: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(reqs))
	}

	if err := s.DuplicateRequest("api", 0); err != nil {
		t.Fatalf("DuplicateRequest: %v", err)
	}
	reqs, _ = s.LoadRequests("api")
	if len(reqs) != 3 || reqs[2].Name != "Get user copy" {
		t.Fatalf("unexpected requests after duplicate: %+v", reqs)
	}

	if err := s.DeleteRequest("api", 1); err != nil {
		t.Fatalf("DeleteRequest: %v", err)
	}
	reqs, _ = s.LoadRequests("api")
	if len(reqs) != 2 || reqs[0].Name != "Get user" || reqs[1].Name != "Get user copy" {
		t.Fatalf("unexpected requests after delete: %+v", reqs)
	}
}

func TestListEmptyDirDoesNotError(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "does-not-exist"))
	cols, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if cols != nil {
		t.Fatalf("expected nil collections, got %+v", cols)
	}
}

// Package collection manages lazycurl request collections: one .http file
// per collection, listed and edited under a config directory.
package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

const fileExt = ".http"

// Collection identifies a single .http collection file.
type Collection struct {
	Name string // file name without extension
	Path string // absolute path to the .http file
}

// Store reads and writes collections rooted at a single directory
// (typically ~/.config/lazycurl/collections).
type Store struct {
	Dir string
}

// NewStore returns a Store rooted at dir.
func NewStore(dir string) *Store {
	return &Store{Dir: dir}
}

// List enumerates all collections (`.http` files) in the store directory,
// sorted by name.
func (s *Store) List() ([]Collection, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing collections: %w", err)
	}

	var collections []Collection
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), fileExt) {
			continue
		}
		collections = append(collections, Collection{
			Name: strings.TrimSuffix(e.Name(), fileExt),
			Path: filepath.Join(s.Dir, e.Name()),
		})
	}
	sort.Slice(collections, func(i, j int) bool { return collections[i].Name < collections[j].Name })
	return collections, nil
}

// path returns the .http file path for a collection name.
func (s *Store) path(name string) string {
	return filepath.Join(s.Dir, name+fileExt)
}

// LoadRequests reads and parses all requests in a collection.
func (s *Store) LoadRequests(name string) ([]httpfile.Request, error) {
	data, err := os.ReadFile(s.path(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading collection %q: %w", name, err)
	}
	return httpfile.Parse(string(data)), nil
}

// SaveRequests serializes and writes all requests for a collection,
// overwriting the existing file (or creating it).
func (s *Store) SaveRequests(name string, requests []httpfile.Request) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("creating collections dir: %w", err)
	}
	content := httpfile.Serialize(requests)
	if err := os.WriteFile(s.path(name), []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing collection %q: %w", name, err)
	}
	return nil
}

// CreateCollection creates a new, empty collection file. It is a no-op if
// the collection already exists.
func (s *Store) CreateCollection(name string) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("creating collections dir: %w", err)
	}
	p := s.path(name)
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	return os.WriteFile(p, nil, 0o644)
}

// CreateRequest appends a new request to the end of a collection.
func (s *Store) CreateRequest(name string, req httpfile.Request) error {
	requests, err := s.LoadRequests(name)
	if err != nil {
		return err
	}
	requests = append(requests, req)
	return s.SaveRequests(name, requests)
}

// DuplicateRequest appends a copy of the request at index to the end of the
// collection, with " copy" appended to its name.
func (s *Store) DuplicateRequest(name string, index int) error {
	requests, err := s.LoadRequests(name)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(requests) {
		return fmt.Errorf("duplicating request: index %d out of range (0..%d)", index, len(requests)-1)
	}
	dup := requests[index]
	dup.Headers = append([]httpfile.KV(nil), dup.Headers...)
	dup.Name = dup.Name + " copy"
	requests = append(requests, dup)
	return s.SaveRequests(name, requests)
}

// DeleteRequest removes the request at index from the collection.
func (s *Store) DeleteRequest(name string, index int) error {
	requests, err := s.LoadRequests(name)
	if err != nil {
		return err
	}
	if index < 0 || index >= len(requests) {
		return fmt.Errorf("deleting request: index %d out of range (0..%d)", index, len(requests)-1)
	}
	requests = append(requests[:index], requests[index+1:]...)
	return s.SaveRequests(name, requests)
}

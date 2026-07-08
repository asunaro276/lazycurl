// Package environment manages lazycurl environment files (dev/staging/prod
// style variable sets) and {{variable}} expansion of requests.
package environment

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const envFileSuffix = ".env.json"

// Store reads environment files and tracks each collection's active
// environment.
//
// Layout:
//
//	<EnvRoot>/<collection>/<name>.env.json   - variable definitions
//	<StatePath>                              - JSON: collection -> active env name
type Store struct {
	EnvRoot   string
	StatePath string
}

// NewStore returns a Store rooted at envRoot, persisting active-environment
// selection to statePath.
func NewStore(envRoot, statePath string) *Store {
	return &Store{EnvRoot: envRoot, StatePath: statePath}
}

func (s *Store) collectionDir(collection string) string {
	return filepath.Join(s.EnvRoot, collection)
}

// List returns the names of environments defined for a collection, sorted.
func (s *Store) List(collection string) ([]string, error) {
	entries, err := os.ReadDir(s.collectionDir(collection))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing environments: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), envFileSuffix) {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), envFileSuffix))
	}
	sort.Strings(names)
	return names, nil
}

// Load reads the variable set for a named environment within a collection.
func (s *Store) Load(collection, name string) (map[string]string, error) {
	path := filepath.Join(s.collectionDir(collection), name+envFileSuffix)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading environment %q: %w", name, err)
	}
	var vars map[string]string
	if err := json.Unmarshal(data, &vars); err != nil {
		return nil, fmt.Errorf("parsing environment %q: %w", name, err)
	}
	return vars, nil
}

// Save writes the variable set for a named environment within a collection.
func (s *Store) Save(collection, name string, vars map[string]string) error {
	dir := s.collectionDir(collection)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating environment dir: %w", err)
	}
	data, err := json.MarshalIndent(vars, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding environment %q: %w", name, err)
	}
	path := filepath.Join(dir, name+envFileSuffix)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing environment %q: %w", name, err)
	}
	return nil
}

type state struct {
	ActiveByCollection map[string]string `json:"activeByCollection"`
}

func (s *Store) loadState() (state, error) {
	data, err := os.ReadFile(s.StatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return state{ActiveByCollection: map[string]string{}}, nil
		}
		return state{}, fmt.Errorf("reading state: %w", err)
	}
	var st state
	if err := json.Unmarshal(data, &st); err != nil {
		return state{}, fmt.Errorf("parsing state: %w", err)
	}
	if st.ActiveByCollection == nil {
		st.ActiveByCollection = map[string]string{}
	}
	return st, nil
}

func (s *Store) saveState(st state) error {
	if err := os.MkdirAll(filepath.Dir(s.StatePath), 0o755); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	return os.WriteFile(s.StatePath, data, 0o644)
}

// ActiveEnvironment returns the currently active environment name for a
// collection, or "" if none has been selected.
func (s *Store) ActiveEnvironment(collection string) (string, error) {
	st, err := s.loadState()
	if err != nil {
		return "", err
	}
	return st.ActiveByCollection[collection], nil
}

// SetActiveEnvironment persists the active environment for a collection.
func (s *Store) SetActiveEnvironment(collection, name string) error {
	st, err := s.loadState()
	if err != nil {
		return err
	}
	st.ActiveByCollection[collection] = name
	return s.saveState(st)
}

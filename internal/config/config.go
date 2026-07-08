// Package config manages lazycurl's on-disk configuration directory
// (~/.config/lazycurl/) and startup environment checks.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Dirs holds the resolved paths of lazycurl's configuration directory tree.
type Dirs struct {
	Root        string // ~/.config/lazycurl
	Collections string // ~/.config/lazycurl/collections
}

// Resolve returns the lazycurl config directory paths without creating them.
func Resolve() (Dirs, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Dirs{}, fmt.Errorf("resolving user config dir: %w", err)
	}
	root := filepath.Join(base, "lazycurl")
	return Dirs{
		Root:        root,
		Collections: filepath.Join(root, "collections"),
	}, nil
}

// EnsureDirs creates the lazycurl config directory structure if it does not
// already exist. Existing directories and files are left untouched.
func EnsureDirs() (Dirs, error) {
	dirs, err := Resolve()
	if err != nil {
		return Dirs{}, err
	}
	if err := os.MkdirAll(dirs.Collections, 0o755); err != nil {
		return Dirs{}, fmt.Errorf("creating collections dir: %w", err)
	}
	return dirs, nil
}

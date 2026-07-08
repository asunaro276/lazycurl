package curlexec

import (
	"context"
	"testing"
)

func TestCheckVersion(t *testing.T) {
	result, err := CheckVersion(context.Background())
	if err != nil {
		t.Fatalf("CheckVersion: %v", err)
	}
	if !result.Found {
		t.Fatal("expected curl to be found")
	}
	if result.Version == "" {
		t.Fatal("expected a parsed version string")
	}
	if !result.MeetsMinVerion {
		t.Errorf("expected installed curl to meet min version %s, got %s", MinVersion, result.Version)
	}
}

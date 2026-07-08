package curlexec

import (
	"strings"
	"testing"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

func TestBuildArgsBasicGet(t *testing.T) {
	req := httpfile.Request{Method: "GET", URL: "https://example.com/users/1"}
	args := buildArgs(req, "", "/tmp/h", "/tmp/o")

	assertContainsSeq(t, args, []string{"-X", "GET"})
	assertContainsSeq(t, args, []string{"-D", "/tmp/h"})
	assertContainsSeq(t, args, []string{"-o", "/tmp/o"})
	assertContainsSeq(t, args, []string{"-w", "%{json}"})
	if !contains(args, "-L") {
		t.Errorf("expected -L (redirects followed by default), args=%v", args)
	}
	if contains(args, "--data-binary") {
		t.Errorf("did not expect --data-binary for bodyless request, args=%v", args)
	}
}

func TestBuildArgsPragmas(t *testing.T) {
	req := httpfile.Request{
		Method: "GET",
		URL:    "https://example.com",
		Pragmas: httpfile.Pragmas{
			Insecure:   true,
			Timeout:    "5s",
			NoRedirect: true,
		},
	}
	args := buildArgs(req, "", "/tmp/h", "/tmp/o")

	if !contains(args, "-k") {
		t.Errorf("expected -k for @insecure, args=%v", args)
	}
	assertContainsSeq(t, args, []string{"--max-time", "5"})
	if contains(args, "-L") {
		t.Errorf("did not expect -L when @no-redirect set, args=%v", args)
	}
}

func TestBuildArgsBodyAndHeaders(t *testing.T) {
	req := httpfile.Request{
		Method: "POST",
		URL:    "https://example.com/items",
		Headers: []httpfile.KV{
			{Key: "Content-Type", Value: "application/json", Enabled: true},
			{Key: "X-Disabled", Value: "nope", Enabled: false},
		},
		Body: `{"a":1}`,
	}
	args := buildArgs(req, "/tmp/body-in", "/tmp/h", "/tmp/o")

	assertContainsSeq(t, args, []string{"-H", "Content-Type: application/json"})
	if contains(args, "X-Disabled: nope") {
		t.Errorf("disabled header should not be included, args=%v", args)
	}
	assertContainsSeq(t, args, []string{"--data-binary", "@/tmp/body-in"})
}

func TestBuildArgsAuth(t *testing.T) {
	req := httpfile.Request{
		Method: "GET",
		URL:    "https://example.com",
		Auth:   httpfile.Auth{Type: httpfile.AuthBearer, Token: "abc123"},
	}
	args := buildArgs(req, "", "/tmp/h", "/tmp/o")
	assertContainsSeq(t, args, []string{"-H", "Authorization: Bearer abc123"})
}

func TestShellQuote(t *testing.T) {
	argv := []string{"curl", "-X", "GET", "https://example.com/a b", "-H", "X: it's fine"}
	got := ShellQuote(argv)
	want := `curl -X GET 'https://example.com/a b' -H 'X: it'\''s fine'`
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func contains(args []string, s string) bool {
	for _, a := range args {
		if a == s {
			return true
		}
	}
	return false
}

func assertContainsSeq(t *testing.T, args []string, seq []string) {
	t.Helper()
	joined := strings.Join(args, "\x00")
	target := strings.Join(seq, "\x00")
	if !strings.Contains(joined, target) {
		t.Errorf("expected args to contain sequence %v, got %v", seq, args)
	}
}

package curlexec

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// fakeRunner simulates curl by writing to the -D/-o files named in argv and
// returning a canned %{json} write-out on stdout, without spawning any
// process. This is the mock used to unit test Executor per task 5.6.
type fakeRunner struct {
	statusCode int
	respBody   string
	exitCode   int
	err        error
}

func (f *fakeRunner) Run(ctx context.Context, argv []string) ([]byte, int, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	if f.exitCode != 0 {
		return nil, f.exitCode, nil
	}
	headerFile := argAfter(argv, "-D")
	outFile := argAfter(argv, "-o")
	_ = os.WriteFile(headerFile, []byte(fmt.Sprintf("HTTP/1.1 %d OK\r\nContent-Type: text/plain\r\n\r\n", f.statusCode)), 0o600)
	_ = os.WriteFile(outFile, []byte(f.respBody), 0o600)
	stdout := []byte(fmt.Sprintf(`{"http_code":%d,"time_total":0.25}`, f.statusCode))
	return stdout, 0, nil
}

func argAfter(argv []string, flag string) string {
	for i, a := range argv {
		if a == flag && i+1 < len(argv) {
			return argv[i+1]
		}
	}
	return ""
}

func TestExecutorWithMockRunner(t *testing.T) {
	e := NewExecutorWithRunner(&fakeRunner{statusCode: 200, respBody: "hello"})
	resp, err := e.Execute(context.Background(), httpfile.Request{Method: "GET", URL: "https://example.com"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if string(resp.Body) != "hello" {
		t.Errorf("expected body 'hello', got %q", resp.Body)
	}
	if resp.TimeTotal != 250*time.Millisecond {
		t.Errorf("expected TimeTotal 250ms, got %v", resp.TimeTotal)
	}
}

func TestExecutorNonZeroExitMapsToHumanError(t *testing.T) {
	e := NewExecutorWithRunner(&fakeRunner{exitCode: 7})
	_, err := e.Execute(context.Background(), httpfile.Request{Method: "GET", URL: "https://example.com"})
	if err == nil {
		t.Fatal("expected error for exit code 7")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 7 || !strings.Contains(exitErr.Message, "接続") {
		t.Errorf("unexpected exit error: %+v", exitErr)
	}
}

func TestExecutorRunnerErrorPropagates(t *testing.T) {
	sentinel := fmt.Errorf("binary not found")
	e := NewExecutorWithRunner(&fakeRunner{err: sentinel})
	_, err := e.Execute(context.Background(), httpfile.Request{Method: "GET", URL: "https://example.com"})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestExecutorRealCurl runs a genuine curl subprocess against an
// httptest.Server to verify the whole path end-to-end (argv construction,
// temp files, header/body separation, write-out JSON parsing).
func TestExecutorRealCurl(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	e := NewExecutor()
	req := httpfile.Request{
		Method: "POST",
		URL:    srv.URL + "/things",
		Headers: []httpfile.KV{
			{Key: "Content-Type", Value: "application/json", Enabled: true},
		},
		Body: `{"name":"widget"}`,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := e.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}
	if string(resp.Body) != `{"ok":true}` {
		t.Errorf("unexpected body: %q", resp.Body)
	}
	found := false
	for _, h := range resp.Headers {
		if strings.EqualFold(h.Key, "X-Test") && h.Value == "yes" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected X-Test header in response, got %+v", resp.Headers)
	}
	if resp.TimeTotal <= 0 {
		t.Errorf("expected positive TimeTotal, got %v", resp.TimeTotal)
	}

	yanked := ShellQuote(resp.Argv)
	if !strings.Contains(yanked, "curl") || !strings.Contains(yanked, "-X POST") || !strings.Contains(yanked, srv.URL+"/things") {
		t.Errorf("unexpected yanked command: %q", yanked)
	}
}

func TestExecutorRealCurlCancellation(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-block
	}))
	defer func() {
		close(block)
		srv.Close()
	}()

	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := e.Execute(ctx, httpfile.Request{Method: "GET", URL: srv.URL})
	if err == nil {
		t.Fatal("expected error when context is cancelled mid-request")
	}
}

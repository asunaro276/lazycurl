package curlexec

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// mockServerBaseURL is the base URL of the singleton mockserver container
// started in TestMain, shared by every E2E test case in this package.
var mockServerBaseURL string

// Fixed credentials the mockserver's /auth/basic and /auth/bearer endpoints
// accept (testing/mockserver/handlers.go). Duplicated here rather than
// imported since root-module Go code must not import the mockserver module
// (they communicate over HTTP only).
const (
	mockBasicAuthUser = "lazycurl"
	mockBasicAuthPass = "curl-e2e"
	mockBearerToken   = "lazycurl-e2e-token"
)

// TestMain builds and starts a single testing/mockserver container (from
// testing/mockserver/Dockerfile) for the whole test binary, so individual
// E2E test cases don't each pay container startup cost. It terminates the
// container once all tests (E2E and unit) have finished.
func TestMain(m *testing.M) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context:    "../../testing/mockserver",
			Dockerfile: "Dockerfile",
		},
		ExposedPorts: []string{"8080/tcp"},
		WaitingFor:   wait.ForHTTP("/status/200").WithPort("8080/tcp").WithStartupTimeout(60 * time.Second),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting mockserver container: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "terminating mockserver container: %v\n", err)
		}
	}()

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "getting mockserver host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "getting mockserver mapped port: %v\n", err)
		os.Exit(1)
	}
	mockServerBaseURL = fmt.Sprintf("http://%s:%s", host, port.Port())

	os.Exit(m.Run())
}

// TestE2ERedirectFollowed verifies that -L (the default, no @no-redirect)
// causes the real curl subprocess to follow the mockserver's redirect
// chain down to the final 200 response.
func TestE2ERedirectFollowed(t *testing.T) {
	e := NewExecutor()
	req := httpfile.Request{
		Method: "GET",
		URL:    mockServerBaseURL + "/redirect/1",
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200 (redirect should have been followed)", resp.StatusCode)
	}
}

// TestE2ERedirectNotFollowedWithNoRedirectPragma verifies that
// @no-redirect suppresses -L, so curl stops at the first 3xx hop instead of
// following it.
func TestE2ERedirectNotFollowedWithNoRedirectPragma(t *testing.T) {
	e := NewExecutor()
	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/redirect/1",
		Pragmas: httpfile.Pragmas{NoRedirect: true},
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("StatusCode = %d, want 302 (redirect should not have been followed)", resp.StatusCode)
	}
}

// TestE2ETimeoutExpires verifies that @timeout (--max-time) causes curl to
// give up before the mockserver's artificial delay elapses, rather than
// blocking indefinitely.
func TestE2ETimeoutExpires(t *testing.T) {
	e := NewExecutor()
	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/delay/5",
		Pragmas: httpfile.Pragmas{Timeout: "500ms"},
	}

	start := time.Now()
	_, err := e.Execute(context.Background(), req)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	exitErr, ok := err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T: %v", err, err)
	}
	if exitErr.Code != 28 {
		t.Errorf("exit code = %d, want 28 (operation timeout)", exitErr.Code)
	}
	if elapsed >= 5*time.Second {
		t.Errorf("elapsed = %v, should have timed out well before the 5s delay", elapsed)
	}
}

// TestE2EBasicAuth verifies that httpfile.Auth (Basic) is derived into a
// real Authorization header that the mockserver accepts.
func TestE2EBasicAuth(t *testing.T) {
	e := NewExecutor()
	req := httpfile.Request{
		Method: "GET",
		URL:    mockServerBaseURL + "/auth/basic",
		Auth:   httpfile.Auth{Type: httpfile.AuthBasic, Username: mockBasicAuthUser, Password: mockBasicAuthPass},
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

// TestE2EBearerAuth is the Bearer-scheme counterpart of TestE2EBasicAuth.
func TestE2EBearerAuth(t *testing.T) {
	e := NewExecutor()
	req := httpfile.Request{
		Method: "GET",
		URL:    mockServerBaseURL + "/auth/bearer",
		Auth:   httpfile.Auth{Type: httpfile.AuthBearer, Token: mockBearerToken},
	}
	resp, err := e.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
}

// TestE2EStatusAndEchoParsing verifies that the status code (-w
// '%{json}'/-D) and body (-o) come back correctly parsed for both a
// fixed-status endpoint and one that echoes the request back.
func TestE2EStatusAndEchoParsing(t *testing.T) {
	e := NewExecutor()

	t.Run("status", func(t *testing.T) {
		resp, err := e.Execute(context.Background(), httpfile.Request{
			Method: "GET",
			URL:    mockServerBaseURL + "/status/201",
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if resp.StatusCode != 201 {
			t.Errorf("StatusCode = %d, want 201", resp.StatusCode)
		}
	})

	t.Run("echo", func(t *testing.T) {
		resp, err := e.Execute(context.Background(), httpfile.Request{
			Method:  "POST",
			URL:     mockServerBaseURL + "/echo",
			Headers: []httpfile.KV{{Key: "X-Lazycurl", Value: "e2e", Enabled: true}},
			Body:    `{"hello":"world"}`,
		})
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("StatusCode = %d, want 200", resp.StatusCode)
		}
		body := string(resp.Body)
		for _, want := range []string{`"method":"POST"`, `"body":"{\"hello\":\"world\"}"`, `"X-Lazycurl":["e2e"]`} {
			if !strings.Contains(body, want) {
				t.Errorf("echoed body missing %q: %s", want, body)
			}
		}
	})
}

// TestE2EStreamMultipleChunks verifies that a @stream request against
// /stream delivers more than one Chunk event over a real HTTP connection
// before the terminal Done event, and that the natural-completion body
// matches the concatenation of every chunk received.
func TestE2EStreamMultipleChunks(t *testing.T) {
	e := NewExecutor()
	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/stream?chunks=4&interval=100",
		Pragmas: httpfile.Pragmas{Stream: true},
	}

	events := e.ExecuteStreaming(context.Background(), req)

	var chunkCount int
	var body []byte
	var done *StreamDone
	for ev := range events {
		if ev.Chunk != nil {
			chunkCount++
			body = append(body, ev.Chunk...)
		}
		if ev.Done != nil {
			done = ev.Done
		}
	}

	if chunkCount < 2 {
		t.Errorf("chunkCount = %d, want >= 2 (should arrive incrementally)", chunkCount)
	}
	if done == nil {
		t.Fatal("expected a terminal Done event")
	}
	if done.Err != nil {
		t.Errorf("Done.Err = %v, want nil", done.Err)
	}
	if string(done.Response.Body) != string(body) {
		t.Errorf("Done.Response.Body = %q, want concatenation of received chunks %q", done.Response.Body, body)
	}
}

// TestE2EStreamCancelMidway verifies that cancelling the context passed to
// ExecuteStreaming after the first chunk arrives (the ctrl-c scenario)
// yields a StreamDone with the partial body received so far and a nil Err
// -- cancellation is a normal early end, not a failure.
func TestE2EStreamCancelMidway(t *testing.T) {
	e := NewExecutor()
	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/stream?chunks=6&interval=500",
		Pragmas: httpfile.Pragmas{Stream: true},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := e.ExecuteStreaming(ctx, req)

	var chunkCount int
	var partialBody []byte
	var done *StreamDone
	for ev := range events {
		if ev.Chunk != nil {
			chunkCount++
			partialBody = append(partialBody, ev.Chunk...)
			if chunkCount == 2 {
				cancel()
			}
		}
		if ev.Done != nil {
			done = ev.Done
		}
	}

	if chunkCount >= 6 {
		t.Fatalf("chunkCount = %d, want < 6 (cancellation should have cut the stream short)", chunkCount)
	}
	if done == nil {
		t.Fatal("expected a terminal Done event")
	}
	if done.Err != nil {
		t.Errorf("Done.Err = %v, want nil (cancellation is a normal early end)", done.Err)
	}
	if string(done.Response.Body) != string(partialBody) {
		t.Errorf("Done.Response.Body = %q, want partial body received before cancellation %q", done.Response.Body, partialBody)
	}
}

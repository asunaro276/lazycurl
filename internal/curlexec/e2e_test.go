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

// mockServerBaseURL is the base URL of the shared testing/mockserver
// container, started once in TestMain and reused by every E2E test in this
// package (see openspec/changes/curlexec-e2e-mock-server/design.md,
// "自動テストはtestcontainers-goでDockerfileから都度ビルドし...").
var mockServerBaseURL string

// The credentials below must match the constants in
// testing/mockserver/handlers.go (basicAuthUser/basicAuthPassword/
// bearerAuthToken). They are duplicated here rather than imported because
// root-module code must never import the mockserver module (it is a
// separate Go module communicating only over HTTP).
const (
	mockBasicAuthUser     = "lazycurl"
	mockBasicAuthPassword = "secret"
	mockBearerAuthToken   = "lazycurl-token"
)

func TestMain(m *testing.M) {
	os.Exit(runE2EMain(m))
}

// runE2EMain is factored out of TestMain so deferred cleanup (container
// termination) always runs, even though TestMain itself must call os.Exit.
func runE2EMain(m *testing.M) int {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "../../testing/mockserver",
				Dockerfile: "Dockerfile",
			},
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor:   wait.ForHTTP("/status/200").WithPort("8080/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "curlexec E2E: starting mockserver container: %v\n", err)
		return 1
	}
	defer func() {
		tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(tctx); err != nil {
			fmt.Fprintf(os.Stderr, "curlexec E2E: terminating mockserver container: %v\n", err)
		}
	}()

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "curlexec E2E: resolving mockserver host: %v\n", err)
		return 1
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "curlexec E2E: resolving mockserver port: %v\n", err)
		return 1
	}
	mockServerBaseURL = fmt.Sprintf("http://%s:%s", host, port.Port())

	return m.Run()
}

// TestE2ERedirectFollowedByDefault verifies that, absent @no-redirect,
// buildArgs' `-L` makes a real curl subprocess follow the mockserver's
// redirect chain to the final 200.
func TestE2ERedirectFollowedByDefault(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := e.Execute(ctx, httpfile.Request{Method: "GET", URL: mockServerBaseURL + "/redirect/2"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected final status 200 after following redirects, got %d", resp.StatusCode)
	}
}

// TestE2ERedirectNotFollowedWithNoRedirectPragma verifies that
// @no-redirect suppresses `-L`, leaving curl on the first 3xx hop.
func TestE2ERedirectNotFollowedWithNoRedirectPragma(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/redirect/1",
		Pragmas: httpfile.Pragmas{NoRedirect: true},
	}
	resp, err := e.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("expected unfollowed status 302, got %d", resp.StatusCode)
	}
}

// TestE2ETimeoutErrorsWithinDeadline verifies that @timeout's `--max-time`
// aborts a real curl subprocess against a slow mockserver response well
// before the response would otherwise arrive.
func TestE2ETimeoutErrorsWithinDeadline(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/delay/5",
		Pragmas: httpfile.Pragmas{Timeout: "300ms"},
	}

	start := time.Now()
	_, err := e.Execute(ctx, req)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error")
	}
	if elapsed >= 5*time.Second {
		t.Errorf("expected curl to time out well before the 5s delay, took %v", elapsed)
	}
}

// TestE2EBasicAuthHeaderSentCorrectly verifies that a Basic Auth
// httpfile.Request is derived into a working `Authorization` header.
func TestE2EBasicAuthHeaderSentCorrectly(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := httpfile.Request{
		Method: "GET",
		URL:    mockServerBaseURL + "/auth/basic",
		Auth:   httpfile.Auth{Type: httpfile.AuthBasic, Username: mockBasicAuthUser, Password: mockBasicAuthPassword},
	}
	resp, err := e.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for correct basic auth, got %d", resp.StatusCode)
	}
}

// TestE2EBearerAuthHeaderSentCorrectly verifies that a Bearer Auth
// httpfile.Request is derived into a working `Authorization` header.
func TestE2EBearerAuthHeaderSentCorrectly(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := httpfile.Request{
		Method: "GET",
		URL:    mockServerBaseURL + "/auth/bearer",
		Auth:   httpfile.Auth{Type: httpfile.AuthBearer, Token: mockBearerAuthToken},
	}
	resp, err := e.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for correct bearer auth, got %d", resp.StatusCode)
	}
}

// TestE2EStatusHeadersAndBodyParsedCorrectly exercises /status/{code} and
// /echo to verify that a real curl invocation's `-w '%{json}'` status,
// `-D` headers, and `-o` body are all parsed correctly by Execute.
func TestE2EStatusHeadersAndBodyParsedCorrectly(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	statusResp, err := e.Execute(ctx, httpfile.Request{Method: "GET", URL: mockServerBaseURL + "/status/201"})
	if err != nil {
		t.Fatalf("Execute /status/201: %v", err)
	}
	if statusResp.StatusCode != 201 {
		t.Errorf("expected status 201, got %d", statusResp.StatusCode)
	}

	echoReq := httpfile.Request{
		Method: "POST",
		URL:    mockServerBaseURL + "/echo",
		Headers: []httpfile.KV{
			{Key: "X-Lazycurl-Test", Value: "yes", Enabled: true},
		},
		Body: `{"hello":"world"}`,
	}
	echoResp, err := e.Execute(ctx, echoReq)
	if err != nil {
		t.Fatalf("Execute /echo: %v", err)
	}
	if echoResp.StatusCode != 200 {
		t.Errorf("expected 200 from /echo, got %d", echoResp.StatusCode)
	}
	body := string(echoResp.Body)
	if !strings.Contains(body, `"method":"POST"`) {
		t.Errorf("expected echoed method POST, got body %q", body)
	}
	if !strings.Contains(body, `{\"hello\":\"world\"}`) {
		t.Errorf("expected echoed request body, got body %q", body)
	}
	if !strings.Contains(body, "X-Lazycurl-Test") {
		t.Errorf("expected echoed custom header, got body %q", body)
	}
}

// TestE2EStreamingDeliversMultipleChunksThenCompletes verifies that
// ExecuteStreaming, driving a real curl subprocess in `-N`/`-o -` mode
// against /stream, delivers more than one StreamEvent{Chunk} before the
// terminal Done, and that the natural-completion body matches the full
// concatenation of chunks sent by the server.
func TestE2EStreamingDeliversMultipleChunksThenCompletes(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/stream?chunks=3&interval=100",
		Pragmas: httpfile.Pragmas{Stream: true},
	}

	ch := e.ExecuteStreaming(ctx, req)

	var chunkCount int
	var body []byte
	var done *StreamDone
	for event := range ch {
		if event.Done != nil {
			done = event.Done
			continue
		}
		chunkCount++
		body = append(body, event.Chunk...)
	}

	if done == nil {
		t.Fatal("expected a terminal Done event")
	}
	if chunkCount < 2 {
		t.Errorf("expected at least 2 chunk events before Done, got %d", chunkCount)
	}
	if done.Err != nil {
		t.Errorf("expected nil Err on natural completion, got %v", done.Err)
	}
	if string(done.Response.Body) != string(body) {
		t.Errorf("expected StreamDone.Response.Body to match concatenated chunks %q, got %q", body, done.Response.Body)
	}
	if done.Response.Body == nil || string(done.Response.Body) != "chunk-0\nchunk-1\nchunk-2\n" {
		t.Errorf("unexpected final streamed body: %q", done.Response.Body)
	}
}

// TestE2EStreamingCancellationYieldsPartialBody verifies that cancelling
// the context mid-stream (the ctrl-c scenario) stops curl early and yields
// a StreamDone with only the chunks received so far and a nil Err, per
// streaming.go's "cancellation is a normal early end, not a failure"
// contract.
func TestE2EStreamingCancellationYieldsPartialBody(t *testing.T) {
	e := NewExecutor()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	streamCtx, streamCancel := context.WithCancel(ctx)
	defer streamCancel()

	req := httpfile.Request{
		Method:  "GET",
		URL:     mockServerBaseURL + "/stream?chunks=10&interval=300",
		Pragmas: httpfile.Pragmas{Stream: true},
	}

	ch := e.ExecuteStreaming(streamCtx, req)

	var chunkCount int
	var body []byte
	var done *StreamDone
	for event := range ch {
		if event.Done != nil {
			done = event.Done
			continue
		}
		chunkCount++
		body = append(body, event.Chunk...)
		if chunkCount == 2 {
			streamCancel()
		}
	}

	if done == nil {
		t.Fatal("expected a terminal Done event")
	}
	if done.Err != nil {
		t.Errorf("expected nil Err on cancellation (early end is not a failure), got %v", done.Err)
	}
	if chunkCount >= 10 {
		t.Errorf("expected cancellation to cut the stream short of all 10 chunks, got %d chunks", chunkCount)
	}
	if string(done.Response.Body) != string(body) {
		t.Errorf("expected StreamDone.Response.Body to match received chunks %q, got %q", body, done.Response.Body)
	}
}

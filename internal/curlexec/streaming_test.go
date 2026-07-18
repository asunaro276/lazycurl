package curlexec

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// sseTestServer starts an httptest.Server that flushes a few chunks with a
// short delay between them (simulating a slow SSE feed) and returns its URL.
// The server is closed automatically via t.Cleanup.
func sseTestServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected ResponseWriter to support flushing")
		}
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 3; i++ {
			fmt.Fprintf(w, "data: chunk-%d\n\n", i)
			flusher.Flush()
			time.Sleep(50 * time.Millisecond)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// fakeStreamRunner simulates a streaming curl process: it writes the header
// file (as -D would) and delivers chunks one at a time on the chunks
// channel, without spawning any process.
type fakeStreamRunner struct {
	statusCode int
	chunks     [][]byte
	exitCode   int
	err        error
	cancelable bool // if true, stop delivering chunks and block until ctx is done
}

func (f *fakeStreamRunner) Run(ctx context.Context, argv []string, chunks chan<- []byte) (int, error) {
	defer close(chunks)

	headerFile := argAfter(argv, "-D")
	_ = os.WriteFile(headerFile, []byte(fmt.Sprintf("HTTP/1.1 %d OK\r\nContent-Type: text/event-stream\r\n\r\n", f.statusCode)), 0o600)

	if f.err != nil {
		return 0, f.err
	}

	for _, c := range f.chunks {
		select {
		case chunks <- c:
		case <-ctx.Done():
			return 0, nil
		}
	}

	if f.cancelable {
		<-ctx.Done()
		return 0, nil
	}

	return f.exitCode, nil
}

func TestExecuteStreamingDeliversChunksThenDone(t *testing.T) {
	e := NewExecutorWithRunners(nil, &fakeStreamRunner{
		statusCode: 200,
		chunks:     [][]byte{[]byte("data: hello\n\n"), []byte("data: world\n\n")},
	})
	req := httpfile.Request{Method: "GET", URL: "https://example.com/events", Pragmas: httpfile.Pragmas{Stream: true}}

	ch := e.ExecuteStreaming(context.Background(), req)

	var gotChunks [][]byte
	var done *StreamDone
	for ev := range ch {
		if ev.Done != nil {
			done = ev.Done
			continue
		}
		gotChunks = append(gotChunks, ev.Chunk)
	}

	if len(gotChunks) != 2 {
		t.Fatalf("expected 2 chunk events, got %d", len(gotChunks))
	}
	if string(gotChunks[0]) != "data: hello\n\n" || string(gotChunks[1]) != "data: world\n\n" {
		t.Errorf("unexpected chunks: %q", gotChunks)
	}
	if done == nil {
		t.Fatal("expected a terminal Done event")
	}
	if done.Err != nil {
		t.Errorf("unexpected error: %v", done.Err)
	}
	if done.Response == nil {
		t.Fatal("expected a Response in the Done event")
	}
	if done.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", done.Response.StatusCode)
	}
	if string(done.Response.Body) != "data: hello\n\ndata: world\n\n" {
		t.Errorf("expected concatenated body, got %q", done.Response.Body)
	}
}

func TestExecuteStreamingCancellationKeepsPartialBodyWithoutError(t *testing.T) {
	e := NewExecutorWithRunners(nil, &fakeStreamRunner{
		statusCode: 200,
		chunks:     [][]byte{[]byte("chunk1")},
		cancelable: true,
	})
	req := httpfile.Request{Method: "GET", URL: "https://example.com/events", Pragmas: httpfile.Pragmas{Stream: true}}

	ctx, cancel := context.WithCancel(context.Background())
	ch := e.ExecuteStreaming(ctx, req)

	// Consume the first chunk, then cancel mid-stream.
	ev := <-ch
	if ev.Chunk == nil || string(ev.Chunk) != "chunk1" {
		t.Fatalf("expected first chunk, got %+v", ev)
	}
	cancel()

	var done *StreamDone
	for ev := range ch {
		if ev.Done != nil {
			done = ev.Done
		}
	}

	if done == nil {
		t.Fatal("expected a terminal Done event after cancellation")
	}
	if done.Err != nil {
		t.Errorf("expected cancellation to not be surfaced as an error, got %v", done.Err)
	}
	if string(done.Response.Body) != "chunk1" {
		t.Errorf("expected partial body 'chunk1', got %q", done.Response.Body)
	}
}

func TestExecuteStreamingNonZeroExitMapsToHumanError(t *testing.T) {
	e := NewExecutorWithRunners(nil, &fakeStreamRunner{statusCode: 0, exitCode: 7})
	req := httpfile.Request{Method: "GET", URL: "https://example.com/events", Pragmas: httpfile.Pragmas{Stream: true}}

	ch := e.ExecuteStreaming(context.Background(), req)
	var done *StreamDone
	for ev := range ch {
		if ev.Done != nil {
			done = ev.Done
		}
	}

	if done == nil {
		t.Fatal("expected a terminal Done event")
	}
	exitErr, ok := done.Err.(*ExitError)
	if !ok {
		t.Fatalf("expected *ExitError, got %T: %v", done.Err, done.Err)
	}
	if exitErr.Code != 7 {
		t.Errorf("expected exit code 7, got %d", exitErr.Code)
	}
}

func TestExecuteStreamingRealCurlSSE(t *testing.T) {
	// End-to-end check against a real curl subprocess and a server that
	// dribbles out an SSE-shaped body over time, verifying chunks actually
	// arrive incrementally rather than all at once at the end.
	e := NewExecutor()
	req := httpfile.Request{Method: "GET", URL: sseTestServer(t), Pragmas: httpfile.Pragmas{Stream: true}}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch := e.ExecuteStreaming(ctx, req)

	var chunkCount int
	var done *StreamDone
	for ev := range ch {
		if ev.Done != nil {
			done = ev.Done
			continue
		}
		chunkCount++
	}

	if chunkCount < 2 {
		t.Errorf("expected at least 2 separate chunk events from a dribbling server, got %d", chunkCount)
	}
	if done == nil || done.Err != nil {
		t.Fatalf("expected a successful Done event, got %+v", done)
	}
	if done.Response.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", done.Response.StatusCode)
	}
	if done.Response.TimeTotal <= 0 {
		t.Errorf("expected positive TimeTotal, got %v", done.Response.TimeTotal)
	}
}

func TestExecuteStreamingRealCurlCancellation(t *testing.T) {
	// End-to-end check that ctrl-c-style ctx cancellation of a real curl
	// subprocess mid-stream still yields the partial body received so far,
	// with no error -- the scenario task 6.1 asks to confirm manually.
	first := make(chan struct{})
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "data: first\n\n")
		flusher.Flush()
		close(first)
		<-block
	}))
	defer func() {
		close(block)
		srv.Close()
	}()

	e := NewExecutor()
	req := httpfile.Request{Method: "GET", URL: srv.URL, Pragmas: httpfile.Pragmas{Stream: true}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := e.ExecuteStreaming(ctx, req)

	var body []byte
	var done *StreamDone
	cancelled := false
	for ev := range ch {
		if ev.Done != nil {
			done = ev.Done
			continue
		}
		body = append(body, ev.Chunk...)
		if !cancelled {
			<-first
			cancel()
			cancelled = true
		}
	}

	if done == nil {
		t.Fatal("expected a terminal Done event")
	}
	if done.Err != nil {
		t.Errorf("expected cancellation to not be surfaced as an error, got %v", done.Err)
	}
	if string(body) != "data: first\n\n" {
		t.Errorf("expected partial body from before cancellation, got %q", body)
	}
}

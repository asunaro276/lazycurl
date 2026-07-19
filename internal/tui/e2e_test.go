package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/asunaro276/lazycurl/internal/collection"
	"github.com/asunaro276/lazycurl/internal/curlexec"
	"github.com/asunaro276/lazycurl/internal/environment"
	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// mockServerBaseURL is the base URL of the shared testing/mockserver
// container, started once in TestMain and reused by every TUI E2E test in
// this package. internal/curlexec starts its own, independent instance of
// the same container (see its e2e_test.go) since Go builds one test binary
// per package (design.md, "内部curlexecと内部tuiはそれぞれ別々にコンテナを1つ起動する").
var mockServerBaseURL string

func TestMain(m *testing.M) {
	os.Exit(runE2EMain(m))
}

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
		fmt.Fprintf(os.Stderr, "tui E2E: starting mockserver container: %v\n", err)
		return 1
	}
	defer func() {
		tctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := container.Terminate(tctx); err != nil {
			fmt.Fprintf(os.Stderr, "tui E2E: terminating mockserver container: %v\n", err)
		}
	}()

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui E2E: resolving mockserver host: %v\n", err)
		return 1
	}
	port, err := container.MappedPort(ctx, "8080/tcp")
	if err != nil {
		fmt.Fprintf(os.Stderr, "tui E2E: resolving mockserver port: %v\n", err)
		return 1
	}
	mockServerBaseURL = fmt.Sprintf("http://%s:%s", host, port.Port())

	return m.Run()
}

// newE2EApp constructs an App wired to a fresh, temp-dir-backed collection
// containing the given pre-seeded requests, and a real curlexec.Executor
// (genuine curl subprocesses), matching what tui-e2e-testing/spec.md
// requires: keyboard-driven navigation over real HTTP against the
// mockserver container, never a fakeRunner and never direct model field
// pokes for pragmas the inline form can't represent (like @stream).
func newE2EApp(t *testing.T, requests []httpfile.Request) *App {
	t.Helper()
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))

	if err := colStore.CreateCollection("e2e"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := colStore.SaveRequests("e2e", requests); err != nil {
		t.Fatalf("SaveRequests: %v", err)
	}

	app, err := New(colStore, envStore, curlexec.NewExecutor())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return app
}

// selectFirstRequestAndSend sends the key sequence that a user would type
// to move to the Collections panel, select the collection's first (and
// only, in these tests) request, load it into the [0] Request panel, and
// send it.
func selectFirstRequestAndSend(tm *teatest.TestModel) {
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")}) // -> Collections panel
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // header -> first request row
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})                     // load request into [0] Request
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})                     // send
}

func containsBytes(haystack []byte, needle string) bool {
	return len(haystack) > 0 && indexOfBytes(string(haystack), needle) >= 0
}

func indexOfBytes(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// historyEntryRendered reports whether the History panel's
// "HH:MM:SS <status> <name>" summary line for a completed send is present
// in bts. Matching this exact triplet (rather than a bare status-code
// substring) avoids false positives from status-code-like digits that
// happen to appear in the mockserver's dynamically assigned port number.
func historyEntryRendered(bts []byte, statusCode int, name string) bool {
	pattern := fmt.Sprintf(`\d\d:\d\d:\d\d %d %s`, statusCode, regexp.QuoteMeta(name))
	return regexp.MustCompile(pattern).Match(bts)
}

// TestE2ETUISendRequestUpdatesResponsePanel drives Collections navigation,
// Request selection, and send entirely through key input (per
// tui-e2e-testing/spec.md's "TUI E2Eテストはteatestで実際のtea.Programを駆動する"
// and "送信結果はResponseパネルの描画に反映される"), verifying via teatest's
// rendered terminal output that the mockserver's real HTTP response reaches
// the Response panel.
func TestE2ETUISendRequestUpdatesResponsePanel(t *testing.T) {
	app := newE2EApp(t, []httpfile.Request{
		{Name: "Status", Method: "GET", URL: mockServerBaseURL + "/status/200"},
	})

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
	selectFirstRequestAndSend(tm)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return historyEntryRendered(bts, 200, "Status")
	}, teatest.WithDuration(10*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

// TestE2ETUIStreamingShowsPartialBodyThenCompletes verifies that, for an
// `@stream` request sent to /stream, the Response panel shows partial
// chunk content before the send completes, and the full concatenated body
// is present once it finishes naturally.
func TestE2ETUIStreamingShowsPartialBodyThenCompletes(t *testing.T) {
	app := newE2EApp(t, []httpfile.Request{
		{
			Name:    "Stream",
			Method:  "GET",
			URL:     mockServerBaseURL + "/stream?chunks=3&interval=200",
			Pragmas: httpfile.Pragmas{Stream: true},
		},
	})

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
	selectFirstRequestAndSend(tm)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return containsBytes(bts, "chunk-0") && !containsBytes(bts, "chunk-2")
	}, teatest.WithDuration(3*time.Second))

	// Let the send complete naturally, then confirm the full body arrived.
	time.Sleep(1 * time.Second)
	final, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !containsBytes(final, "chunk-2") {
		t.Errorf("expected all chunks present after natural completion, got %q", final)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

// TestE2ETUIStreamingCtrlCConfirmsPartialBodyInHistory verifies that
// cancelling an in-flight `@stream` send with ctrl-c confirms a History
// entry containing only the chunks received up to that point, rather than
// the full body.
func TestE2ETUIStreamingCtrlCConfirmsPartialBodyInHistory(t *testing.T) {
	app := newE2EApp(t, []httpfile.Request{
		{
			Name:    "Stream",
			Method:  "GET",
			URL:     mockServerBaseURL + "/stream?chunks=10&interval=300",
			Pragmas: httpfile.Pragmas{Stream: true},
		},
	})

	tm := teatest.NewTestModel(t, app, teatest.WithInitialTermSize(120, 40))
	selectFirstRequestAndSend(tm)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return containsBytes(bts, "chunk-0")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC}) // cancel mid-stream, not quit
	time.Sleep(500 * time.Millisecond)

	captured, err := io.ReadAll(tm.Output())
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if containsBytes(captured, "chunk-9") {
		t.Errorf("expected cancellation to cut the stream short of all 10 chunks, got %q", captured)
	}
	if !containsBytes(captured, "Stream") {
		t.Errorf("expected the cancelled send to be confirmed in History, got %q", captured)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
}

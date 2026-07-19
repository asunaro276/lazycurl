package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// mockServerBaseURL is the base URL of the singleton mockserver container
// started in TestMain, shared by every E2E test case in this package. It is
// independent of internal/curlexec's own container (built and started
// separately, per package, per the design's singleton-per-test-binary
// pattern).
var mockServerBaseURL string

// TestMain builds and starts a single testing/mockserver container for the
// whole internal/tui test binary, mirroring internal/curlexec's TestMain.
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

// newE2ETestModel wires up a fresh App (real curl via curlexec.NewExecutor,
// no fake Runner) with a single-request "e2e" collection, and drives it
// through teatest as a real tea.Program.
func newE2ETestModel(t *testing.T, width, height int, req httpfile.Request) *teatest.TestModel {
	t.Helper()
	dir := t.TempDir()
	colStore := collection.NewStore(filepath.Join(dir, "collections"))
	envStore := environment.NewStore(filepath.Join(dir, "env"), filepath.Join(dir, "state.json"))
	executor := curlexec.NewExecutor()

	if err := colStore.CreateCollection("e2e"); err != nil {
		t.Fatalf("CreateCollection: %v", err)
	}
	if err := colStore.CreateRequest("e2e", req); err != nil {
		t.Fatalf("CreateRequest: %v", err)
	}

	app, err := New(colStore, envStore, executor)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return teatest.NewTestModel(t, app, teatest.WithInitialTermSize(width, height))
}

// selectAndSendFirstRequest drives the key sequence for: jump to the [2]
// Collections panel, move the cursor onto the (only) request row of the
// already-expanded "e2e" collection, load it into [0] Request (which also
// moves focus there), then send it.
func selectAndSendFirstRequest(tm *teatest.TestModel) {
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})
}

// TestE2EKeyDrivenSendRendersResponse covers key-driven selection/sending
// (Collections -> Request -> send) reaching the mockserver over a real curl
// subprocess, and the response's status code being rendered into the
// Response panel.
func TestE2EKeyDrivenSendRendersResponse(t *testing.T) {
	req := httpfile.Request{Name: "Status", Method: "GET", URL: mockServerBaseURL + "/status/200"}
	tm := newE2ETestModel(t, 120, 40, req)
	defer func() {
		tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	}()

	selectAndSendFirstRequest(tm)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "200")
	}, teatest.WithDuration(10*time.Second))
}

const streamPhrase = "the quick brown fox jumps over the lazy dog "

// TestE2EStreamPartialRenderBeforeCompletion covers @stream's incremental
// Response panel rendering. teatest.WaitFor drains the program's output as
// it polls, so a condition met by an early WaitFor call can't be
// re-inspected afterwards -- instead this asserts "partial, not yet
// complete" via timing: chunks=3 at 400ms apart takes ~1200ms end to end,
// so seeing the streamed body within 700ms proves the Response panel
// updated before the send (and its eventual History confirmation) could
// possibly have finished.
func TestE2EStreamPartialRenderBeforeCompletion(t *testing.T) {
	req := httpfile.Request{
		Name:    "Stream",
		Method:  "GET",
		URL:     mockServerBaseURL + "/stream?chunks=3&interval=400",
		Pragmas: httpfile.Pragmas{Stream: true},
	}
	tm := newE2ETestModel(t, 300, 40, req)
	defer func() {
		tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	}()

	selectAndSendFirstRequest(tm)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), streamPhrase)
	}, teatest.WithDuration(700*time.Millisecond), teatest.WithCheckInterval(20*time.Millisecond))

	// Eventually the send completes naturally and the request is
	// confirmed into History with its full response.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "Stream") && strings.Contains(string(bts), "200")
	}, teatest.WithDuration(10*time.Second))
}

// TestE2EStreamCtrlCConfirmsPartialHistory covers cancelling an
// in-progress @stream send (the ctrl-c scenario): the send is cut short,
// and a History entry is confirmed well before all chunks could have
// arrived naturally, proving the cancellation -- not natural completion --
// is what produced it.
func TestE2EStreamCtrlCConfirmsPartialHistory(t *testing.T) {
	req := httpfile.Request{
		Name:    "StreamCancel",
		Method:  "GET",
		URL:     mockServerBaseURL + "/stream?chunks=6&interval=500",
		Pragmas: httpfile.Pragmas{Stream: true},
	}
	tm := newE2ETestModel(t, 300, 40, req)
	defer func() {
		tm.Quit()
		tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))
	}()

	selectAndSendFirstRequest(tm)

	// Wait for the first chunk to arrive, then cancel -- well before all 6
	// chunks (~3s) would have arrived naturally.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), streamPhrase)
	}, teatest.WithDuration(2*time.Second))
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	// The History panel (always rendered, regardless of focus) should
	// confirm the cut-short request quickly -- well under the ~3s a full,
	// uncancelled run would take.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "StreamCancel")
	}, teatest.WithDuration(1500*time.Millisecond))
}

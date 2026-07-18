package curlexec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// Executor sends variable-expanded requests via a curl subprocess.
type Executor struct {
	runner       Runner
	streamRunner StreamRunner
}

// NewExecutor returns an Executor backed by the real curl binary.
func NewExecutor() *Executor {
	return &Executor{runner: execRunner{}, streamRunner: execStreamRunner{}}
}

// NewExecutorWithRunner returns an Executor backed by a custom Runner,
// primarily for unit testing without invoking a real subprocess. Its
// ExecuteStreaming path is left without a StreamRunner; use
// NewExecutorWithRunners to also test streaming.
func NewExecutorWithRunner(r Runner) *Executor {
	return &Executor{runner: r}
}

// NewExecutorWithRunners returns an Executor backed by custom Runner and
// StreamRunner implementations, for unit testing both Execute and
// ExecuteStreaming without invoking a real subprocess.
func NewExecutorWithRunners(r Runner, sr StreamRunner) *Executor {
	return &Executor{runner: r, streamRunner: sr}
}

// BuildArgv returns the curl argv that would be used to execute req,
// without running it. Used both by Execute and by the "yank as curl
// command" feature.
func BuildArgv(req httpfile.Request, bodyFile, headerFile, outFile string) []string {
	return buildArgs(req, bodyFile, headerFile, outFile)
}

// Execute runs req as a curl subprocess and returns the structured
// response. req must already have {{variable}} expansion applied - Execute
// does not perform variable substitution.
//
// If curl exits with a non-zero code, the returned error is an *ExitError
// carrying a human-readable message.
func (e *Executor) Execute(ctx context.Context, req httpfile.Request) (*Response, error) {
	tmpDir, err := os.MkdirTemp("", "lazycurl-")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	headerFile := tmpDir + "/headers"
	outFile := tmpDir + "/body"

	var bodyFile string
	if req.Body != "" {
		bodyFile = tmpDir + "/request-body"
		if err := os.WriteFile(bodyFile, []byte(req.Body), 0o600); err != nil {
			return nil, fmt.Errorf("writing request body temp file: %w", err)
		}
	}

	argv := buildArgs(req, bodyFile, headerFile, outFile)

	stdout, exitCode, err := e.runner.Run(ctx, argv)
	if err != nil {
		return nil, err
	}
	if exitCode != 0 {
		return nil, newExitError(exitCode)
	}

	statusCode, headers, timeTotal := parseWriteOut(stdout)

	headerRaw, err := os.ReadFile(headerFile)
	if err != nil {
		return nil, fmt.Errorf("reading response headers: %w", err)
	}
	parsedStatus, parsedHeaders := parseHeaderFile(headerRaw)
	if parsedStatus != 0 {
		statusCode = parsedStatus
	}
	if len(parsedHeaders) > 0 {
		headers = parsedHeaders
	}

	body, err := os.ReadFile(outFile)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return &Response{
		StatusCode: statusCode,
		Headers:    headers,
		Body:       body,
		TimeTotal:  timeTotal,
		Argv:       argv,
	}, nil
}

type writeOutJSON struct {
	HTTPCode     int     `json:"http_code"`
	ResponseCode int     `json:"response_code"`
	TimeTotal    float64 `json:"time_total"`
}

func parseWriteOut(stdout []byte) (statusCode int, headers []httpfile.KV, timeTotal time.Duration) {
	var w writeOutJSON
	if err := json.Unmarshal(stdout, &w); err != nil {
		return 0, nil, 0
	}
	statusCode = w.HTTPCode
	if statusCode == 0 {
		statusCode = w.ResponseCode
	}
	return statusCode, nil, time.Duration(w.TimeTotal * float64(time.Second))
}

package curlexec

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// StreamRunner abstracts a streaming subprocess so ExecuteStreaming can be
// unit tested without invoking a real curl binary. Implementations run
// argv[0] with argv[1:] as arguments, delivering each chunk of stdout to
// chunks as it is read and closing chunks once no more data will arrive
// (whether the process exited normally or ctx was cancelled).
//
// err/exitCode follow the same convention as Runner.Run: a non-nil err
// means the process could not be started or was killed for a reason other
// than a normal exit, in which case exitCode is meaningless.
type StreamRunner interface {
	Run(ctx context.Context, argv []string, chunks chan<- []byte) (exitCode int, err error)
}

// execStreamRunner is the production StreamRunner backed by os/exec,
// reading curl's stdout pipe incrementally.
type execStreamRunner struct{}

func (execStreamRunner) Run(ctx context.Context, argv []string, chunks chan<- []byte) (int, error) {
	defer close(chunks)

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	if err := cmd.Start(); err != nil {
		return 0, err
	}

	buf := make([]byte, 4096)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			chunks <- chunk
		}
		if readErr != nil {
			break
		}
	}

	waitErr := cmd.Wait()
	if waitErr == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(waitErr, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return 0, waitErr
}

// StreamDone is the terminal event of a streaming execution: the fully
// assembled Response (whatever body arrived before completion/cancellation)
// and, if the execution genuinely failed (as opposed to being cancelled),
// the error.
type StreamDone struct {
	Response *Response
	Err      error
}

// StreamEvent is delivered on the channel returned by ExecuteStreaming:
// either a body chunk (Chunk != nil) or the terminal event (Done != nil,
// sent exactly once as the last event before the channel closes).
type StreamEvent struct {
	Chunk []byte
	Done  *StreamDone
}

// ExecuteStreaming runs req (which must have Pragmas.Stream set) as a curl
// subprocess and streams its response body back incrementally, unlike
// Execute which blocks until the full body has been written to a temp file.
// req must already have {{variable}} expansion applied.
//
// Response.TimeTotal is measured on the Go side (elapsed wall time from
// start to completion/cancellation) rather than parsed from curl's
// `-w '%{json}'`, since that write-out is never produced when the process
// is killed by ctx cancellation.
//
// If ctx is cancelled mid-stream, the returned StreamDone carries the body
// received so far with Err == nil -- cancellation is treated as a normal
// (if early) end of the stream, not a failure.
func (e *Executor) ExecuteStreaming(ctx context.Context, req httpfile.Request) <-chan StreamEvent {
	out := make(chan StreamEvent)

	go func() {
		defer close(out)

		tmpDir, err := os.MkdirTemp("", "lazycurl-")
		if err != nil {
			out <- StreamEvent{Done: &StreamDone{Err: fmt.Errorf("creating temp dir: %w", err)}}
			return
		}
		defer os.RemoveAll(tmpDir)

		headerFile := tmpDir + "/headers"

		var bodyFile string
		if req.Body != "" {
			bodyFile = tmpDir + "/request-body"
			if err := os.WriteFile(bodyFile, []byte(req.Body), 0o600); err != nil {
				out <- StreamEvent{Done: &StreamDone{Err: fmt.Errorf("writing request body temp file: %w", err)}}
				return
			}
		}

		argv := buildArgs(req, bodyFile, headerFile, "")

		chunks := make(chan []byte)
		type result struct {
			exitCode int
			err      error
		}
		resultCh := make(chan result, 1)

		start := time.Now()
		go func() {
			exitCode, runErr := e.streamRunner.Run(ctx, argv, chunks)
			resultCh <- result{exitCode, runErr}
		}()

		var body []byte
		for chunk := range chunks {
			body = append(body, chunk...)
			out <- StreamEvent{Chunk: chunk}
		}
		res := <-resultCh
		elapsed := time.Since(start)

		var statusCode int
		var headers []httpfile.KV
		if headerRaw, readErr := os.ReadFile(headerFile); readErr == nil {
			statusCode, headers = parseHeaderFile(headerRaw)
		}

		var respErr error
		if ctx.Err() == nil {
			if res.err != nil {
				respErr = res.err
			} else if res.exitCode != 0 {
				respErr = newExitError(res.exitCode)
			}
		}

		out <- StreamEvent{Done: &StreamDone{
			Response: &Response{
				StatusCode: statusCode,
				Headers:    headers,
				Body:       body,
				TimeTotal:  elapsed,
				Argv:       argv,
			},
			Err: respErr,
		}}
	}()

	return out
}

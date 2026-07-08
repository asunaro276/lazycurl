package curlexec

import (
	"context"
	"errors"
	"os/exec"
)

// Runner abstracts subprocess execution so Executor can be unit tested
// without invoking a real curl binary. Implementations run argv[0] with
// argv[1:] as arguments, capture stdout (curl's `-w '%{json}'` write-out),
// and report curl's exit code.
//
// A non-nil err indicates the process could not be started or was killed
// for a reason other than a normal (possibly non-zero) exit - e.g. binary
// not found, or ctx cancellation/timeout. In that case exitCode is
// meaningless.
type Runner interface {
	Run(ctx context.Context, argv []string) (stdout []byte, exitCode int, err error)
}

// execRunner is the production Runner backed by os/exec.
type execRunner struct{}

func (execRunner) Run(ctx context.Context, argv []string) ([]byte, int, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	stdout, err := cmd.Output()
	if err == nil {
		return stdout, 0, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return stdout, exitErr.ExitCode(), nil
	}
	// Not started (binary missing), or killed by ctx cancellation.
	return nil, 0, err
}

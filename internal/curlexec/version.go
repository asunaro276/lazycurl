package curlexec

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

// MinVersion is the minimum curl version required for `-w '%{json}'` support.
const MinVersion = "7.70"

var versionRe = regexp.MustCompile(`curl (\d+)\.(\d+)\.(\d+)`)

// VersionCheckResult describes the outcome of checking the local curl binary.
type VersionCheckResult struct {
	Found          bool
	Version        string
	MeetsMinVerion bool
}

// CheckVersion runs `curl --version` and reports whether curl is installed
// and whether it meets MinVersion (required for `-w '%{json}'`).
func CheckVersion(ctx context.Context) (VersionCheckResult, error) {
	out, err := exec.CommandContext(ctx, "curl", "--version").Output()
	if err != nil {
		if isNotFound(err) {
			return VersionCheckResult{}, fmt.Errorf("curl バイナリが見つかりません。curl をインストールしてください: %w", err)
		}
		return VersionCheckResult{}, fmt.Errorf("curl --version の実行に失敗しました: %w", err)
	}

	m := versionRe.FindSubmatch(out)
	if m == nil {
		return VersionCheckResult{Found: true}, nil
	}
	major, _ := strconv.Atoi(string(m[1]))
	minor, _ := strconv.Atoi(string(m[2]))

	meets := major > 7 || (major == 7 && minor >= 70)
	return VersionCheckResult{
		Found:          true,
		Version:        fmt.Sprintf("%d.%d.%s", major, minor, string(m[3])),
		MeetsMinVerion: meets,
	}, nil
}

func isNotFound(err error) bool {
	var execErr *exec.Error
	if eerr, ok := err.(*exec.Error); ok {
		execErr = eerr
	}
	return execErr != nil && execErr.Err == exec.ErrNotFound
}

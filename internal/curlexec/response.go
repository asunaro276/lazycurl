package curlexec

import (
	"strconv"
	"strings"
	"time"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// Response is the structured result of a curl execution: status/timing
// metadata plus separately captured headers and body.
type Response struct {
	StatusCode int
	Headers    []httpfile.KV
	Body       []byte
	TimeTotal  time.Duration
	Argv       []string // full curl argv used to produce this response (for yank)
}

// parseHeaderFile parses the raw content written by curl's -D option. When
// redirects are followed (-L), curl appends one status-line+headers block
// per hop; only the final block (the actual response served) is returned.
func parseHeaderFile(raw []byte) (statusCode int, headers []httpfile.KV) {
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	blocks := splitHeaderBlocks(text)
	if len(blocks) == 0 {
		return 0, nil
	}
	last := blocks[len(blocks)-1]

	lines := strings.Split(strings.Trim(last, "\n"), "\n")
	if len(lines) == 0 {
		return 0, nil
	}
	statusCode = parseStatusLine(lines[0])

	for _, line := range lines[1:] {
		if line == "" {
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		headers = append(headers, httpfile.KV{
			Key:     strings.TrimSpace(line[:idx]),
			Value:   strings.TrimSpace(line[idx+1:]),
			Enabled: true,
		})
	}
	return statusCode, headers
}

// splitHeaderBlocks splits curl's -D output (one or more status-line+header
// blocks separated by blank lines, one block per redirect hop) into
// individual non-empty blocks.
func splitHeaderBlocks(text string) []string {
	rawBlocks := strings.Split(text, "\n\n")
	var blocks []string
	for _, b := range rawBlocks {
		if strings.TrimSpace(b) != "" {
			blocks = append(blocks, b)
		}
	}
	return blocks
}

func parseStatusLine(line string) int {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	code, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0
	}
	return code
}

package httpfile

import (
	"encoding/base64"
	"strings"
)

// Parse reads a full .http collection file and returns the requests it
// contains. Each request is a `###`-delimited block. Unknown `# @pragma`
// lines are ignored so that files written by newer/older versions of
// lazycurl remain loadable.
func Parse(content string) []Request {
	lines := splitLines(content)

	var requests []Request
	var cur *Request
	var bodyLines []string
	inBody := false

	flush := func() {
		if cur == nil {
			return
		}
		cur.Body = strings.TrimRight(strings.Join(bodyLines, "\n"), "\n")
		extractAuth(cur)
		requests = append(requests, *cur)
		cur = nil
		bodyLines = nil
		inBody = false
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "###") {
			flush()
			name := strings.TrimSpace(strings.TrimPrefix(trimmed, "###"))
			cur = &Request{Name: name}
			continue
		}
		if cur == nil {
			// Content outside any ### block is ignored.
			continue
		}

		if inBody {
			bodyLines = append(bodyLines, line)
			continue
		}

		if cur.Method == "" {
			if trimmed == "" {
				continue
			}
			if strings.HasPrefix(trimmed, "#") {
				parsePragmaOrSkip(cur, trimmed)
				continue
			}
			method, url := splitRequestLine(trimmed)
			cur.Method = method
			cur.URL = url
			continue
		}

		// Header section.
		if trimmed == "" {
			inBody = true
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			// A disabled header is stored as `# Key: Value`; a pragma is
			// `# @name ...`. Anything else is a plain comment and ignored.
			rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "#"))
			if strings.HasPrefix(rest, "@") {
				parsePragmaOrSkip(cur, trimmed)
				continue
			}
			if key, val, ok := splitHeaderLine(rest); ok {
				cur.Headers = append(cur.Headers, KV{Key: key, Value: val, Enabled: false})
			}
			continue
		}
		if key, val, ok := splitHeaderLine(trimmed); ok {
			cur.Headers = append(cur.Headers, KV{Key: key, Value: val, Enabled: true})
		}
	}
	flush()

	return requests
}

func splitLines(content string) []string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(content, "\n")
}

func splitRequestLine(line string) (method, url string) {
	parts := strings.SplitN(line, " ", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(line), ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func splitHeaderLine(line string) (key, value string, ok bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}

// parsePragmaOrSkip parses a `# @name [value]` line into cur.Pragmas.
// Unknown pragma names are silently ignored (forward compatibility).
func parsePragmaOrSkip(cur *Request, line string) {
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
	if !strings.HasPrefix(rest, "@") {
		return
	}
	fields := strings.SplitN(rest, " ", 2)
	name := fields[0]
	var value string
	if len(fields) == 2 {
		value = strings.TrimSpace(fields[1])
	}
	switch name {
	case "@insecure":
		cur.Pragmas.Insecure = true
	case "@timeout":
		cur.Pragmas.Timeout = value
	case "@no-redirect":
		cur.Pragmas.NoRedirect = true
	case "@stream":
		cur.Pragmas.Stream = true
	default:
		// Unknown pragma: ignore.
	}
}

// extractAuth inspects an Authorization header (if any) and populates
// Request.Auth from it, removing the header from Headers so the Auth form
// remains the single source of truth for it.
func extractAuth(r *Request) {
	for i, h := range r.Headers {
		if !strings.EqualFold(h.Key, "Authorization") {
			continue
		}
		value := h.Value
		switch {
		case strings.HasPrefix(value, "Bearer "):
			r.Auth = Auth{Type: AuthBearer, Token: strings.TrimPrefix(value, "Bearer ")}
		case strings.HasPrefix(value, "Basic "):
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, "Basic "))
			if err == nil {
				user, pass, _ := strings.Cut(string(decoded), ":")
				r.Auth = Auth{Type: AuthBasic, Username: user, Password: pass}
			} else {
				continue
			}
		default:
			continue
		}
		r.Headers = append(r.Headers[:i:i], r.Headers[i+1:]...)
		return
	}
}

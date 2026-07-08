package httpfile

import (
	"encoding/base64"
	"strings"
)

// Serialize renders requests back into .http+pragma text, in the same
// format Parse accepts. Requests are separated by a blank line.
func Serialize(requests []Request) string {
	var b strings.Builder
	for i, r := range requests {
		if i > 0 {
			b.WriteString("\n")
		}
		writeRequest(&b, r)
	}
	return b.String()
}

func writeRequest(b *strings.Builder, r Request) {
	b.WriteString("### ")
	b.WriteString(r.Name)
	b.WriteString("\n")

	if r.Pragmas.Insecure {
		b.WriteString("# @insecure\n")
	}
	if r.Pragmas.Timeout != "" {
		b.WriteString("# @timeout ")
		b.WriteString(r.Pragmas.Timeout)
		b.WriteString("\n")
	}
	if r.Pragmas.NoRedirect {
		b.WriteString("# @no-redirect\n")
	}

	b.WriteString(r.Method)
	b.WriteString(" ")
	b.WriteString(r.URL)
	b.WriteString("\n")

	if authHeader, ok := authorizationHeader(r.Auth); ok {
		b.WriteString(authHeader.Key)
		b.WriteString(": ")
		b.WriteString(authHeader.Value)
		b.WriteString("\n")
	}
	for _, h := range r.Headers {
		if !h.Enabled {
			b.WriteString("# ")
		}
		b.WriteString(h.Key)
		b.WriteString(": ")
		b.WriteString(h.Value)
		b.WriteString("\n")
	}

	if r.Body != "" {
		b.WriteString("\n")
		b.WriteString(r.Body)
		b.WriteString("\n")
	}
}

func authorizationHeader(a Auth) (KV, bool) {
	switch a.Type {
	case AuthBearer:
		if a.Token == "" {
			return KV{}, false
		}
		return KV{Key: "Authorization", Value: "Bearer " + a.Token, Enabled: true}, true
	case AuthBasic:
		if a.Username == "" && a.Password == "" {
			return KV{}, false
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(a.Username + ":" + a.Password))
		return KV{Key: "Authorization", Value: "Basic " + encoded, Enabled: true}, true
	default:
		return KV{}, false
	}
}

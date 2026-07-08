package environment

import (
	"regexp"
	"sort"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

var variableRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

// Expand substitutes every `{{variable}}` occurrence in s using vars. Any
// variable name not present in vars is left as-is in the output and also
// reported (deduplicated, sorted) via the returned undefined slice.
func Expand(s string, vars map[string]string) (result string, undefined []string) {
	seen := map[string]bool{}
	result = variableRe.ReplaceAllStringFunc(s, func(match string) string {
		name := variableRe.FindStringSubmatch(match)[1]
		if val, ok := vars[name]; ok {
			return val
		}
		if !seen[name] {
			seen[name] = true
			undefined = append(undefined, name)
		}
		return match
	})
	sort.Strings(undefined)
	return result, undefined
}

// ExpandRequest returns a copy of req with {{variable}} expanded in its URL,
// header values, and body using vars. It also returns the sorted, deduped
// list of variable names referenced but not defined in vars; callers should
// treat a non-empty list as a validation error and refuse to send.
func ExpandRequest(req httpfile.Request, vars map[string]string) (httpfile.Request, []string) {
	undefinedSet := map[string]bool{}
	addUndefined := func(names []string) {
		for _, n := range names {
			undefinedSet[n] = true
		}
	}

	out := req
	var u []string
	out.URL, u = Expand(req.URL, vars)
	addUndefined(u)

	out.Headers = make([]httpfile.KV, len(req.Headers))
	for i, h := range req.Headers {
		nh := h
		nh.Key, u = Expand(h.Key, vars)
		addUndefined(u)
		nh.Value, u = Expand(h.Value, vars)
		addUndefined(u)
		out.Headers[i] = nh
	}

	out.Body, u = Expand(req.Body, vars)
	addUndefined(u)

	switch req.Auth.Type {
	case httpfile.AuthBearer:
		out.Auth.Token, u = Expand(req.Auth.Token, vars)
		addUndefined(u)
	case httpfile.AuthBasic:
		out.Auth.Username, u = Expand(req.Auth.Username, vars)
		addUndefined(u)
		out.Auth.Password, u = Expand(req.Auth.Password, vars)
		addUndefined(u)
	}

	var undefined []string
	for n := range undefinedSet {
		undefined = append(undefined, n)
	}
	sort.Strings(undefined)
	return out, undefined
}

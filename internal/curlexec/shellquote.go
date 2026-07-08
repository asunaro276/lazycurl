package curlexec

import "strings"

// ShellQuote renders an executed argv (e.g. Response.Argv, as returned by
// Executor.Execute) as a single copy-pasteable shell command line, quoting
// any argument that needs it. This is lazycurl's "yank as curl command"
// feature.
func ShellQuote(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = quoteArg(a)
	}
	return strings.Join(parts, " ")
}

func quoteArg(s string) string {
	if s == "" {
		return "''"
	}
	if !strings.ContainsAny(s, " \t\n'\"\\$`!*?[](){}<>|;&~#") {
		return s
	}
	// Single-quote, escaping any embedded single quotes as '\''.
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

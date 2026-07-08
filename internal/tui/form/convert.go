package form

import (
	"net/url"
	"strings"

	"github.com/asunaro276/lazycurl/internal/httpfile"
)

// splitURL separates a URL's base (scheme+host+path) from its query
// parameters, so the Params grid can edit them independently of the URL
// text field. lazycurl URLs may contain {{variable}} placeholders, so this
// avoids full net/url parsing (which would mangle `{{` `}}`) in favor of a
// plain split on the first `?`.
func splitURL(raw string) (base string, params []httpfile.KV) {
	base, query, found := strings.Cut(raw, "?")
	if !found || query == "" {
		return base, nil
	}
	for _, pair := range strings.Split(query, "&") {
		if pair == "" {
			continue
		}
		k, v, _ := strings.Cut(pair, "=")
		k = decodeQueryComponent(k)
		v = decodeQueryComponent(v)
		params = append(params, httpfile.KV{Key: k, Value: v, Enabled: true})
	}
	return base, params
}

func decodeQueryComponent(s string) string {
	if decoded, err := url.QueryUnescape(s); err == nil {
		return decoded
	}
	return s
}

// joinURL reassembles a base URL and Params grid into a single URL string
// with a query string, skipping disabled params.
func joinURL(base string, params []httpfile.KV) string {
	var enabled []httpfile.KV
	for _, p := range params {
		if p.Enabled && p.Key != "" {
			enabled = append(enabled, p)
		}
	}
	if len(enabled) == 0 {
		return base
	}
	var parts []string
	for _, p := range enabled {
		parts = append(parts, url.QueryEscape(p.Key)+"="+url.QueryEscape(p.Value))
	}
	return base + "?" + strings.Join(parts, "&")
}

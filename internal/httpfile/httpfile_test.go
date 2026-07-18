package httpfile

import (
	"reflect"
	"testing"
)

func TestParseBasic(t *testing.T) {
	src := `### Get user (self-signed dev server)
# @insecure
# @timeout 5s
GET {{host}}/users/{{id}}
Authorization: Bearer {{token}}
`
	reqs := Parse(src)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	r := reqs[0]
	if r.Name != "Get user (self-signed dev server)" {
		t.Errorf("unexpected name: %q", r.Name)
	}
	if r.Method != "GET" || r.URL != "{{host}}/users/{{id}}" {
		t.Errorf("unexpected method/url: %q %q", r.Method, r.URL)
	}
	if !r.Pragmas.Insecure || r.Pragmas.Timeout != "5s" {
		t.Errorf("unexpected pragmas: %+v", r.Pragmas)
	}
	if r.Auth.Type != AuthBearer || r.Auth.Token != "{{token}}" {
		t.Errorf("unexpected auth: %+v", r.Auth)
	}
}

func TestParseStreamPragma(t *testing.T) {
	src := `### SSE stream
# @stream
GET {{host}}/events
`
	reqs := Parse(src)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if !reqs[0].Pragmas.Stream {
		t.Errorf("expected Stream pragma to be set, got %+v", reqs[0].Pragmas)
	}
}

func TestParseMultipleRequestsAndUnknownPragma(t *testing.T) {
	src := `### First
GET https://example.com/a

### Second
# @unknown-pragma foo
POST https://example.com/b
Content-Type: application/json

{"k":"v"}

### Third
DELETE https://example.com/c
`
	reqs := Parse(src)
	if len(reqs) != 3 {
		t.Fatalf("expected 3 requests, got %d: %+v", len(reqs), reqs)
	}
	if reqs[1].Method != "POST" || reqs[1].Body != `{"k":"v"}` {
		t.Errorf("unexpected second request: %+v", reqs[1])
	}
	if reqs[1].Pragmas.Insecure || reqs[1].Pragmas.Timeout != "" || reqs[1].Pragmas.NoRedirect {
		t.Errorf("unknown pragma should not set known fields: %+v", reqs[1].Pragmas)
	}
}

func TestDisabledHeaderRoundTrip(t *testing.T) {
	src := `### Req
GET https://example.com
X-Enabled: yes
# X-Disabled: no
`
	reqs := Parse(src)
	if len(reqs) != 1 || len(reqs[0].Headers) != 2 {
		t.Fatalf("unexpected parse result: %+v", reqs)
	}
	if !reqs[0].Headers[0].Enabled || reqs[0].Headers[1].Enabled {
		t.Errorf("unexpected enabled flags: %+v", reqs[0].Headers)
	}

	out := Serialize(reqs)
	reqs2 := Parse(out)
	if len(reqs2) != 1 || len(reqs2[0].Headers) != 2 {
		t.Fatalf("round trip failed: %+v", reqs2)
	}
	if !reflect.DeepEqual(reqs2[0], reqs[0]) {
		t.Errorf("round trip mismatch:\n got: %+v\nwant: %+v", reqs2[0], reqs[0])
	}
}

func TestRoundTripReadWriteRead(t *testing.T) {
	cases := []string{
		`### Simple GET
GET https://api.example.com/health
`,
		`### With headers and body
# @timeout 10s
POST {{host}}/items
Content-Type: application/json
X-Trace: {{trace}}

{
  "name": "widget"
}
`,
		`### Insecure and no-redirect
# @insecure
# @no-redirect
GET https://self-signed.example.com
`,
		`### Bearer auth
GET {{host}}/me
Authorization: Bearer abc123
`,
		`### Basic auth
GET {{host}}/me
Authorization: Basic dXNlcjpwYXNz
`,
		`### Streaming SSE endpoint
# @stream
GET {{host}}/events
`,
	}

	for _, src := range cases {
		first := Parse(src)
		serialized := Serialize(first)
		second := Parse(serialized)

		if len(first) != len(second) {
			t.Fatalf("request count mismatch for %q: %d vs %d", src, len(first), len(second))
		}
		for i := range first {
			if !reflect.DeepEqual(first[i], second[i]) {
				t.Errorf("round trip mismatch for %q:\n first: %+v\nsecond: %+v", src, first[i], second[i])
			}
		}
	}
}

func TestMultiRequestRoundTrip(t *testing.T) {
	src := `### A
GET https://example.com/a

### B
POST https://example.com/b
Content-Type: text/plain

hello world

### C
# @insecure
DELETE https://example.com/c
`
	first := Parse(src)
	second := Parse(Serialize(first))
	if len(first) != 3 || len(second) != 3 {
		t.Fatalf("expected 3 requests each, got %d and %d", len(first), len(second))
	}
	for i := range first {
		if !reflect.DeepEqual(first[i], second[i]) {
			t.Errorf("mismatch at %d:\n first: %+v\nsecond: %+v", i, first[i], second[i])
		}
	}
}

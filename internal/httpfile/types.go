// Package httpfile implements a parser and serializer for lazycurl's
// collection storage format: standard .http request syntax (Method/URL/
// Headers/Body) extended with lightweight `# @pragma` comment lines for
// curl-specific behavior that has no representation in plain HTTP.
package httpfile

// KV is an ordered, individually enable-able key/value pair, used for both
// headers and query params in the request editor.
type KV struct {
	Key     string
	Value   string
	Enabled bool
}

// Pragmas holds curl-specific settings that are not expressible as plain
// HTTP and are instead stored as `# @name` comment lines immediately above
// the request line.
type Pragmas struct {
	Insecure   bool   // @insecure -> curl -k
	Timeout    string // @timeout <duration> -> curl --max-time
	NoRedirect bool   // @no-redirect -> do not pass -L
}

// AuthType identifies which authentication scheme a Request uses.
type AuthType string

const (
	AuthNone   AuthType = "none"
	AuthBasic  AuthType = "basic"
	AuthBearer AuthType = "bearer"
)

// Auth holds credentials for a Request's authentication scheme. Auth is
// realized as an Authorization header at send time / serialization time
// rather than being stored as a literal header in Headers.
type Auth struct {
	Type     AuthType
	Username string // Basic
	Password string // Basic
	Token    string // Bearer
}

// Request is a single HTTP request parsed from (or to be serialized into) a
// `###`-delimited block of a .http collection file.
type Request struct {
	Name    string
	Method  string
	URL     string
	Headers []KV
	Auth    Auth
	Body    string
	Pragmas Pragmas
}

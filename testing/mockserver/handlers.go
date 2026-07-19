package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	basicAuthUser     = "lazycurl"
	basicAuthPassword = "secret"
	bearerAuthToken   = "lazycurl-token"
)

// echoResponse is the JSON body returned by /echo.
type echoResponse struct {
	Method  string      `json:"method"`
	Headers http.Header `json:"headers"`
	Body    string      `json:"body"`
}

// handleEcho reports the request's method, headers, and body back as JSON,
// so tests can assert that curl sent exactly what buildArgs constructed
// (-X, -H, --data-binary).
func handleEcho(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(echoResponse{
		Method:  r.Method,
		Headers: r.Header,
		Body:    string(body),
	})
}

// handleStatus responds with whatever HTTP status code is given in the
// path, verifying curl's `-w '%{json}'` status parsing and `-D` header
// capture.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(r.PathValue("code"))
	if err != nil || code < 100 || code > 599 {
		http.Error(w, "invalid status code", http.StatusBadRequest)
		return
	}
	w.WriteHeader(code)
	fmt.Fprintf(w, "status %d", code)
}

// handleRedirect follows a chain of n redirects before finally responding
// 200, verifying curl's `-L` (and, absent it, `@no-redirect`'s expectation
// that no chain is followed).
func handleRedirect(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 0 {
		http.Error(w, "invalid redirect count", http.StatusBadRequest)
		return
	}
	if n == 0 {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("redirect done"))
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/redirect/%d", n-1), http.StatusFound)
}

// handleDelay waits sec seconds (fractional allowed) before responding,
// verifying curl's `--max-time` (`@timeout`) behavior against a slow
// server. It returns early if the client disconnects (e.g. curl gives up
// on its own timeout) rather than holding the goroutine for the full delay.
func handleDelay(w http.ResponseWriter, r *http.Request) {
	sec, err := strconv.ParseFloat(r.PathValue("sec"), 64)
	if err != nil || sec < 0 {
		http.Error(w, "invalid delay", http.StatusBadRequest)
		return
	}
	select {
	case <-time.After(time.Duration(sec * float64(time.Second))):
	case <-r.Context().Done():
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("delayed"))
}

const defaultStreamChunks = 3
const defaultStreamIntervalMS = 200

// handleStream sends its body as multiple chunks, flushed individually with
// a delay in between, so tests can observe curl's `-N`/`-o -` incremental
// stdout delivery (the `@stream` pragma) rather than a single bulk write.
func handleStream(w http.ResponseWriter, r *http.Request) {
	chunks := queryInt(r, "chunks", defaultStreamChunks)
	interval := queryInt(r, "interval", defaultStreamIntervalMS)
	if chunks < 1 {
		chunks = 1
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	for i := 0; i < chunks; i++ {
		if i > 0 {
			select {
			case <-time.After(time.Duration(interval) * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
		fmt.Fprintf(w, "chunk-%d\n", i)
		flusher.Flush()
	}
}

func queryInt(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

// handleAuthBasic validates a fixed Basic-auth username/password pair,
// verifying that lazycurl's Auth field is correctly derived into an
// `Authorization: Basic ...` header.
func handleAuthBasic(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != basicAuthUser || pass != basicAuthPassword {
		w.Header().Set("WWW-Authenticate", `Basic realm="mockserver"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_, _ = w.Write([]byte("authorized"))
}

// handleAuthBearer validates a fixed bearer token, verifying that
// lazycurl's Auth field is correctly derived into an
// `Authorization: Bearer ...` header.
func handleAuthBearer(w http.ResponseWriter, r *http.Request) {
	const prefix = "Bearer "
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, prefix) || strings.TrimPrefix(authz, prefix) != bearerAuthToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_, _ = w.Write([]byte("authorized"))
}

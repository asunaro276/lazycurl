// Package main implements a standalone HTTP mock server used to exercise
// lazycurl's generated curl argv (internal/curlexec) and TUI against a real
// HTTP connection, both from automated E2E tests (via testcontainers-go) and
// from a developer's terminal (via docker compose).
package main

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	basicAuthUser = "lazycurl"
	basicAuthPass = "curl-e2e"
	bearerToken   = "lazycurl-e2e-token"
)

// NewMux builds the mock server's routing table.
func NewMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/echo", handleEcho)
	mux.HandleFunc("/status/{code}", handleStatus)
	mux.HandleFunc("/redirect/{n}", handleRedirect)
	mux.HandleFunc("/delay/{sec}", handleDelay)
	mux.HandleFunc("/stream", handleStream)
	mux.HandleFunc("/auth/basic", handleAuthBasic)
	mux.HandleFunc("/auth/bearer", handleAuthBearer)
	return mux
}

type echoResponse struct {
	Method  string              `json:"method"`
	Headers map[string][]string `json:"headers"`
	Body    string              `json:"body"`
}

// handleEcho reports the request's method, headers, and body back to the
// caller as JSON, for verifying -X/-H/--data-binary.
func handleEcho(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := echoResponse{
		Method:  r.Method,
		Headers: map[string][]string(r.Header),
		Body:    string(body),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// handleStatus responds with the HTTP status code given in the path, for
// verifying `-w '%{json}'` status-code extraction and the `-D` header file.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	code, err := strconv.Atoi(r.PathValue("code"))
	if err != nil || code < 100 || code > 599 {
		http.Error(w, "invalid status code", http.StatusBadRequest)
		return
	}
	w.WriteHeader(code)
}

// handleRedirect follows a chain of n redirects (/redirect/n -> /redirect/
// n-1 -> ... -> /redirect/0) before returning 200, for verifying -L
// (follow) and @no-redirect (don't follow).
func handleRedirect(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil || n < 0 {
		http.Error(w, "invalid redirect count", http.StatusBadRequest)
		return
	}
	if n == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Redirect(w, r, "/redirect/"+strconv.Itoa(n-1), http.StatusFound)
}

// handleDelay waits the given number of seconds (or until the client
// disconnects) before responding, for verifying --max-time (@timeout).
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
}

const defaultStreamBody = "the quick brown fox jumps over the lazy dog "

// handleStream splits a fixed body into `chunks` pieces (default 4),
// flushing each one `interval` milliseconds apart (default 200ms), for
// verifying @stream's `-N`/`-o -` incremental read path.
func handleStream(w http.ResponseWriter, r *http.Request) {
	chunks := queryInt(r, "chunks", 4)
	if chunks < 1 {
		chunks = 1
	}
	interval := queryInt(r, "interval", 200)
	if interval < 0 {
		interval = 0
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	body := strings.Repeat(defaultStreamBody, chunks)
	pieces := splitIntoN(body, chunks)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	for i, piece := range pieces {
		if i > 0 {
			select {
			case <-time.After(time.Duration(interval) * time.Millisecond):
			case <-r.Context().Done():
				return
			}
		}
		if _, err := w.Write([]byte(piece)); err != nil {
			return
		}
		flusher.Flush()
	}
}

// splitIntoN divides s into n contiguous, roughly equal pieces.
func splitIntoN(s string, n int) []string {
	if n <= 1 {
		return []string{s}
	}
	total := len(s)
	base := total / n
	pieces := make([]string, 0, n)
	start := 0
	for i := 0; i < n; i++ {
		end := start + base
		if i == n-1 {
			end = total
		}
		pieces = append(pieces, s[start:end])
		start = end
	}
	return pieces
}

func queryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// handleAuthBasic validates a fixed Basic-auth credential pair, for
// verifying the Authorization header derived from httpfile.Auth (Basic).
func handleAuthBasic(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != basicAuthUser || pass != basicAuthPass {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// handleAuthBearer validates a fixed bearer token, for verifying the
// Authorization header derived from httpfile.Auth (Bearer).
func handleAuthBearer(w http.ResponseWriter, r *http.Request) {
	const prefix = "Bearer "
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, prefix) || auth[len(prefix):] != bearerToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// basicAuthHeader is a small helper for tests wanting to construct a valid
// Basic Authorization header value without duplicating the encoding logic.
func basicAuthHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}

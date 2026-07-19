package main

import (
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandleEcho(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/echo", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	req.Header.Set("X-Test", "value")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()

	var got echoResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Method != http.MethodPost {
		t.Errorf("Method = %q, want POST", got.Method)
	}
	if got.Body != "hello" {
		t.Errorf("Body = %q, want %q", got.Body, "hello")
	}
	if vs := got.Headers["X-Test"]; len(vs) != 1 || vs[0] != "value" {
		t.Errorf("Headers[X-Test] = %v, want [value]", vs)
	}
}

func TestHandleStatus(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/status/404")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want 404", resp.StatusCode)
	}
}

func TestHandleStatusInvalid(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/status/not-a-number")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want 400", resp.StatusCode)
	}
}

func TestHandleRedirect(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // follow, and record how many hops happened via len(via)
		},
	}

	resp, err := client.Get(srv.URL + "/redirect/2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("final StatusCode = %d, want 200", resp.StatusCode)
	}
	if resp.Request.URL.Path != "/redirect/0" {
		t.Errorf("final path = %q, want /redirect/0", resp.Request.URL.Path)
	}
}

func TestHandleRedirectNoFollow(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Get(srv.URL + "/redirect/2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusFound {
		t.Errorf("StatusCode = %d, want 302 (not followed)", resp.StatusCode)
	}
}

func TestHandleDelay(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	start := time.Now()
	resp, err := http.Get(srv.URL + "/delay/0.2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("elapsed = %v, want >= 200ms", elapsed)
	}
}

func TestHandleStreamChunksAndInterval(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	start := time.Now()
	resp, err := http.Get(srv.URL + "/stream?chunks=3&interval=50")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	reader := bufio.NewReader(resp.Body)
	var reads int
	var body strings.Builder
	buf := make([]byte, 4096)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			reads++
			body.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	elapsed := time.Since(start)

	if reads < 2 {
		t.Errorf("reads = %d, want >= 2 (should arrive as separate chunks)", reads)
	}
	if elapsed < 100*time.Millisecond {
		t.Errorf("elapsed = %v, want >= ~100ms (2 intervals of 50ms)", elapsed)
	}
	if body.Len() == 0 {
		t.Error("expected non-empty streamed body")
	}
}

func TestHandleStreamDefaults(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/stream")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(body) == 0 {
		t.Error("expected non-empty streamed body")
	}
}

func TestHandleAuthBasic(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	cases := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"valid credentials", basicAuthHeader(basicAuthUser, basicAuthPass), http.StatusOK},
		{"wrong password", basicAuthHeader(basicAuthUser, "wrong"), http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/basic", nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestHandleAuthBearer(t *testing.T) {
	srv := httptest.NewServer(NewMux())
	defer srv.Close()

	cases := []struct {
		name       string
		header     string
		wantStatus int
	}{
		{"valid token", "Bearer " + bearerToken, http.StatusOK},
		{"invalid token", "Bearer wrong-token", http.StatusUnauthorized},
		{"missing header", "", http.StatusUnauthorized},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+"/auth/bearer", nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Do: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.wantStatus {
				t.Errorf("StatusCode = %d, want %d", resp.StatusCode, tc.wantStatus)
			}
		})
	}
}

func TestSplitIntoN(t *testing.T) {
	pieces := splitIntoN("abcdefghij", 3)
	if len(pieces) != 3 {
		t.Fatalf("len(pieces) = %d, want 3", len(pieces))
	}
	if strings.Join(pieces, "") != "abcdefghij" {
		t.Errorf("joined pieces = %q, want original string", strings.Join(pieces, ""))
	}
}

func TestQueryInt(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/stream?chunks=5", nil)
	if got := queryInt(req, "chunks", 4); got != 5 {
		t.Errorf("queryInt(chunks) = %d, want 5", got)
	}
	if got := queryInt(req, "interval", 200); got != 200 {
		t.Errorf("queryInt(interval, default) = %d, want 200", got)
	}

	invalid := httptest.NewRequest(http.MethodGet, "/stream?chunks=notanumber", nil)
	if got := queryInt(invalid, "chunks", 4); got != 4 {
		t.Errorf("queryInt(invalid) = %d, want default 4", got)
	}
}

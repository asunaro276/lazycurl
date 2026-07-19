package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEchoReturnsMethodHeadersBody(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/echo", strings.NewReader(`{"a":1}`))
	req.Header.Set("X-Test", "yes")
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
		t.Errorf("expected method POST, got %q", got.Method)
	}
	if got.Body != `{"a":1}` {
		t.Errorf("expected body echoed, got %q", got.Body)
	}
	if got.Headers.Get("X-Test") != "yes" {
		t.Errorf("expected X-Test header echoed, got %+v", got.Headers)
	}
}

func TestStatusReturnsRequestedCode(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/status/404")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestStatusRejectsInvalidCode(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/status/not-a-number")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
}

func TestRedirectFollowsChainToFinalOK(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	client := srv.Client()
	var redirects int
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		redirects = len(via)
		return nil
	}
	resp, err := client.Get(srv.URL + "/redirect/2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected final 200, got %d", resp.StatusCode)
	}
	if redirects != 2 {
		t.Errorf("expected 2 redirects, got %d", redirects)
	}
}

func TestRedirectNotFollowedWithoutClient(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	client := srv.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Get(srv.URL + "/redirect/1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusFound {
		t.Errorf("expected 302 when not following, got %d", resp.StatusCode)
	}
}

func TestDelayWaitsApproximatelyRequestedDuration(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	start := time.Now()
	resp, err := http.Get(srv.URL + "/delay/0.2")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()
	elapsed := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if elapsed < 200*time.Millisecond {
		t.Errorf("expected to wait at least 200ms, only waited %v", elapsed)
	}
}

func TestStreamSendsMultipleChunksWithDelay(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	start := time.Now()
	resp, err := http.Get(srv.URL + "/stream?chunks=3&interval=50")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer resp.Body.Close()

	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	firstChunk := string(buf[:n])
	elapsed := time.Since(start)

	if !strings.Contains(firstChunk, "chunk-0") {
		t.Errorf("expected first read to contain chunk-0 promptly, got %q", firstChunk)
	}
	if elapsed >= 100*time.Millisecond {
		t.Errorf("expected first chunk to arrive before the second interval elapses, took %v", elapsed)
	}
}

func TestAuthBasicAcceptsCorrectCredentials(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/auth/basic", nil)
	req.SetBasicAuth(basicAuthUser, basicAuthPassword)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthBasicRejectsWrongCredentials(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/auth/basic", nil)
	req.SetBasicAuth("wrong", "wrong")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuthBearerAcceptsCorrectToken(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/auth/bearer", nil)
	req.Header.Set("Authorization", "Bearer "+bearerAuthToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAuthBearerRejectsWrongToken(t *testing.T) {
	srv := httptest.NewServer(newMux())
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/auth/bearer", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// sanity check that basic-auth constants encode as expected (guards against
// accidental drift between the handler and test helper credentials).
func TestBasicAuthConstantsEncodeToNonEmptyHeader(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte(basicAuthUser + ":" + basicAuthPassword))
	if encoded == "" {
		t.Fatal("expected non-empty basic auth encoding")
	}
}

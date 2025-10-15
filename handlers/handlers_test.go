package handlers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"encoding/json"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newHTTPResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestHealthHandler_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "OK" {
		t.Fatalf("expected body 'OK', got %q", rr.Body.String())
	}
}

func TestMessageHandler_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/message", nil)
	rr := httptest.NewRecorder()

	MessageHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rr.Code)
	}
}

func TestMessageHandler_Post_OK(t *testing.T) {
	// Stub external HTTP calls used by hosts health checks and LLM
	originalClient := http.DefaultClient
	defer func() { http.DefaultClient = originalClient }()

	http.DefaultClient = &http.Client{
		Timeout: 2 * time.Second,
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			// Simulate unhealthy tele hosts for health checks
			if r.URL.Path == "/health" {
				return newHTTPResponse(http.StatusServiceUnavailable, ""), nil
			}
			// Simulate successful Ollama response
			if r.URL.Host == "ollama:11434" && r.URL.Path == "/api/generate" {
				resp := map[string]string{"response": "changed"}
				buf, _ := json.Marshal(resp)
				return newHTTPResponse(http.StatusOK, string(buf)), nil
			}
			// Default: 404
			return newHTTPResponse(http.StatusNotFound, ""), nil
		}),
	}

	body := bytes.NewBufferString(`{"original_text":"hello world","modified_text":""}`)
	req := httptest.NewRequest(http.MethodPost, "/message", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	MessageHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if rr.Body.String() != "Message received and forwarded (maybe)" {
		t.Fatalf("unexpected body: %q", rr.Body.String())
	}
}

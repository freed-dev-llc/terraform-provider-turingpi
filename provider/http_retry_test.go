package provider

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetryTransport_RetriesOnServiceUnavailable(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryTransport_NoRetryOnClientError(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call (no retry), got %d", calls)
	}
}

func TestRetryTransport_RetriesOnConnectionError(t *testing.T) {
	client := &http.Client{
		Transport: newRetryTransport(http.DefaultTransport),
		Timeout:   10 * time.Second,
	}
	start := time.Now()
	_, err := client.Get("http://localhost:99999")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// base delay 250ms + 500ms between the 3 attempts, so this should take
	// noticeably longer than a single immediate failure would.
	if elapsed < retryBaseDelay {
		t.Errorf("expected retries to add delay, elapsed only %v", elapsed)
	}
}

func TestRetryTransport_NonSeekableBodyNotRetried(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	body := io.MultiReader(bytes.NewReader([]byte("part-one-")), bytes.NewReader([]byte("part-two")))
	req, err := http.NewRequest("POST", server.URL, body)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if req.GetBody != nil {
		t.Fatal("test setup invalid: expected GetBody to be nil for io.MultiReader body")
	}

	client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if calls != 1 {
		t.Errorf("expected exactly 1 call for a non-replayable body, got %d", calls)
	}
}

func TestRetryTransport_ContextCancelledStopsRetry(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
	resp, _ := client.Do(req)
	if resp != nil {
		_ = resp.Body.Close()
	}

	if calls >= retryMaxAttempts {
		t.Errorf("expected cancellation to stop retries before %d attempts, got %d", retryMaxAttempts, calls)
	}
}

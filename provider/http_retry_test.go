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
	resp, err := client.Get(server.URL + "/api/bmc?opt=get&type=power")
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
	resp, err := client.Get(server.URL + "/api/bmc?opt=get&type=info")
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
	_, err := client.Get("http://localhost:99999/api/bmc?opt=get&type=about")
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
	req, err := http.NewRequest("POST", server.URL+"/api/bmc/authenticate", body)
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

func TestRetryTransport_MutatingSetNotRetried(t *testing.T) {
	// The BMC mutates via GET; a transport error does not prove the request
	// never took effect, so sets that are unsafe to repeat get one attempt.
	urls := []string{
		"/api/bmc?opt=set&type=flash&node=1&file=http://example.com/image.raw.xz",
		"/api/bmc?opt=set&type=reset&node=2",
		"/api/bmc?opt=set&type=firmware&length=1024",
		"/api/bmc?opt=set&type=reboot",
		"/api/bmc?opt=set&type=network",
		"/api/bmc?opt=set&type=node_to_msd&node=1",
	}
	for _, u := range urls {
		var calls int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusServiceUnavailable)
		}))

		client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
		resp, err := client.Get(server.URL + u)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", u, err)
		}
		_ = resp.Body.Close()
		server.Close()

		if calls != 1 {
			t.Errorf("%s: expected exactly 1 call, got %d", u, calls)
		}
	}
}

func TestRetryTransport_IdempotentSetRetried(t *testing.T) {
	for _, u := range []string{
		"/api/bmc?opt=set&type=power&node1=1",
		"/api/bmc?opt=set&type=reload",
	} {
		var calls int
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))

		client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
		resp, err := client.Get(server.URL + u)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", u, err)
		}
		status := resp.StatusCode
		_ = resp.Body.Close()
		server.Close()

		if status != http.StatusOK {
			t.Errorf("%s: expected 200 after retries, got %d", u, status)
		}
		if calls != 3 {
			t.Errorf("%s: expected 3 calls, got %d", u, calls)
		}
	}
}

func TestRetryTransport_AuthenticatePostRetriesAndReplaysBody(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
	payload := `{"username":"root","password":"secret"}`
	resp, err := client.Post(server.URL+"/api/bmc/authenticate", "application/json", bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if len(bodies) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(bodies))
	}
	for i, b := range bodies {
		if b != payload {
			t.Errorf("attempt %d: body not replayed intact, got %q", i+1, b)
		}
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
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/api/bmc?opt=get&type=about", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	client := &http.Client{Transport: newRetryTransport(http.DefaultTransport)}
	resp, err := client.Do(req)

	if calls >= retryMaxAttempts {
		t.Errorf("expected cancellation to stop retries before %d attempts, got %d", retryMaxAttempts, calls)
	}
	// Regression for #152: cancellation during backoff must surface as an
	// error, not as a response whose body has already been closed.
	if err == nil {
		t.Fatal("expected an error after cancellation, got nil")
	}
	if resp != nil {
		_ = resp.Body.Close()
		t.Errorf("expected nil response after cancellation, got status %d", resp.StatusCode)
	}
}

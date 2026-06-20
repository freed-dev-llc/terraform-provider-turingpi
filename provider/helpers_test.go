package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckPowerStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"response": [][]interface{}{
				{"node1", float64(1)},
				{"node2", float64(0)},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	if state, err := checkPowerStatus(server.URL, "token", 1); err != nil {
		t.Fatalf("node1: unexpected error: %s", err)
	} else if state != "on" {
		t.Errorf("node1: expected 'on', got %s", state)
	}

	if state, err := checkPowerStatus(server.URL, "token", 2); err != nil {
		t.Fatalf("node2: unexpected error: %s", err)
	} else if state != "off" {
		t.Errorf("node2: expected 'off', got %s", state)
	}
}

func TestCheckPowerStatus_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if _, err := checkPowerStatus(server.URL, "token", 1); err == nil {
		t.Fatal("expected error when BMC returns 500, got nil")
	}
}

func TestTurnOnNode(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := turnOnNode(server.URL, "token", 1); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !strings.Contains(capturedURL, "node1=1") {
		t.Errorf("expected power-on URL to contain 'node1=1', got %s", capturedURL)
	}
}

func TestTurnOffNode(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	if err := turnOffNode(server.URL, "token", 2); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !strings.Contains(capturedURL, "node2=0") {
		t.Errorf("expected power-off URL to contain 'node2=0', got %s", capturedURL)
	}
}

func TestFlashNode_FileNotFound(t *testing.T) {
	// flashNode opens the firmware file before contacting the BMC, so a missing
	// file surfaces as an error without any network call.
	err := flashNode(context.Background(), "https://test.local", "token", 1, "/nonexistent/firmware.img")
	if err == nil {
		t.Fatal("expected error for missing firmware file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to open firmware file") {
		t.Errorf("expected file-open error, got: %s", err)
	}
}

func TestCheckBootStatus_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "GET" {
			t.Errorf("expected GET request, got %s", r.Method)
		}

		// Verify Authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("expected Bearer token in Authorization header, got %s", auth)
		}

		// Verify query parameters
		if r.URL.Query().Get("opt") != "get" {
			t.Errorf("expected opt=get, got %s", r.URL.Query().Get("opt"))
		}
		if r.URL.Query().Get("type") != "uart" {
			t.Errorf("expected type=uart, got %s", r.URL.Query().Get("type"))
		}

		// Return response with login prompt
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Some boot output...\nlogin: "))
	}))
	defer server.Close()

	// Use short timeout since mock server returns immediately
	success, err := checkBootStatus(context.Background(), server.URL, 1, 1, "test-token", "login:")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if !success {
		t.Error("expected success=true when login prompt is found")
	}
}

func TestCheckBootStatus_TokenInHeader(t *testing.T) {
	expectedToken := "my-secret-token"
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("login:"))
	}))
	defer server.Close()

	_, _ = checkBootStatus(context.Background(), server.URL, 1, 1, expectedToken, "login:")

	expectedHeader := "Bearer " + expectedToken
	if capturedAuth != expectedHeader {
		t.Errorf("expected Authorization header '%s', got '%s'", expectedHeader, capturedAuth)
	}
}

func TestCheckBootStatus_NodeInURL(t *testing.T) {
	testCases := []struct {
		node         int
		expectedNode string
	}{
		{1, "1"},
		{2, "2"},
		{3, "3"},
		{4, "4"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("node_%d", tc.node), func(t *testing.T) {
			var capturedNode string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedNode = r.URL.Query().Get("node")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("login:"))
			}))
			defer server.Close()

			_, _ = checkBootStatus(context.Background(), server.URL, tc.node, 1, "token", "login:")

			if capturedNode != tc.expectedNode {
				t.Errorf("expected node=%s in URL, got node=%s", tc.expectedNode, capturedNode)
			}
		})
	}
}

func TestCheckBootStatus_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return response without login prompt
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Booting...\nStill booting..."))
	}))
	defer server.Close()

	// Use very short timeout to speed up test
	// Note: This test will take at least 1 second due to the timeout
	success, err := checkBootStatus(context.Background(), server.URL, 1, 1, "token", "login:")

	if success {
		t.Error("expected success=false on timeout")
	}

	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout reached") {
		t.Errorf("expected timeout error message, got: %s", err.Error())
	}
}

func TestCheckBootStatus_ConnectionError(t *testing.T) {
	// Use invalid URL to simulate connection error
	success, err := checkBootStatus(context.Background(), "http://localhost:99999", 1, 1, "token", "login:")

	if success {
		t.Error("expected success=false on connection error")
	}

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCheckBootStatus_URLConstruction(t *testing.T) {
	var capturedPath string
	var capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("login:"))
	}))
	defer server.Close()

	_, _ = checkBootStatus(context.Background(), server.URL, 2, 1, "token", "login:")

	if capturedPath != "/api/bmc" {
		t.Errorf("expected path /api/bmc, got %s", capturedPath)
	}

	if !strings.Contains(capturedQuery, "opt=get") {
		t.Errorf("expected query to contain opt=get, got %s", capturedQuery)
	}

	if !strings.Contains(capturedQuery, "type=uart") {
		t.Errorf("expected query to contain type=uart, got %s", capturedQuery)
	}

	if !strings.Contains(capturedQuery, "node=2") {
		t.Errorf("expected query to contain node=2, got %s", capturedQuery)
	}
}

func TestCheckBootStatus_LoginPromptVariations(t *testing.T) {
	testCases := []struct {
		name     string
		response string
		expected bool
	}{
		{"login prompt at end", "boot complete\nlogin:", true},
		{"login prompt with space", "boot complete\nlogin: ", true},
		{"login prompt in middle", "stuff\nlogin:\nmore stuff", true},
		{"no login prompt", "still booting...", false},
		{"empty response", "", false},
		{"partial match", "logging in...", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()

			success, _ := checkBootStatus(context.Background(), server.URL, 1, 1, "token", "login:")

			if success != tc.expected {
				t.Errorf("expected success=%v for response '%s', got %v", tc.expected, tc.response, success)
			}
		})
	}
}

func TestCheckBootStatus_CustomPattern(t *testing.T) {
	testCases := []struct {
		name     string
		response string
		pattern  string
		expected bool
	}{
		{"talos ready pattern", "[talos] machine is running and ready", "machine is running and ready", true},
		{"talos boot sequence done", "[talos] boot sequence: done", "boot sequence: done", true},
		{"custom pattern match", "System initialized successfully", "initialized successfully", true},
		{"pattern not found", "still booting...", "machine is running and ready", false},
		{"wrong pattern", "login:", "machine is running and ready", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tc.response))
			}))
			defer server.Close()

			success, _ := checkBootStatus(context.Background(), server.URL, 1, 1, "token", tc.pattern)

			if success != tc.expected {
				t.Errorf("expected success=%v for pattern '%s' in response '%s', got %v", tc.expected, tc.pattern, tc.response, success)
			}
		})
	}
}

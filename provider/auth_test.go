package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate_Success(t *testing.T) {
	expectedToken := "test-token-12345"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// Verify request path
		if r.URL.Path != "/api/bmc/authenticate" {
			t.Errorf("expected path /api/bmc/authenticate, got %s", r.URL.Path)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		body, _ := io.ReadAll(r.Body)
		var reqData map[string]string
		_ = json.Unmarshal(body, &reqData)

		if reqData["username"] != "testuser" {
			t.Errorf("expected username 'testuser', got %s", reqData["username"])
		}
		if reqData["password"] != "testpass" {
			t.Errorf("expected password 'testpass', got %s", reqData["password"])
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"id": expectedToken})
	}))
	defer server.Close()

	token, err := authenticate(server.URL, "testuser", "testpass")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if token != expectedToken {
		t.Errorf("expected token %s, got %s", expectedToken, token)
	}
}

func TestAuthenticate_InvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	_, err := authenticate(server.URL, "baduser", "badpass")
	if err == nil {
		t.Fatal("expected error for invalid credentials, got nil")
	}

	expectedErr := `BMC rejected credentials for "baduser" (status 401):  - check TURINGPI_USERNAME/TURINGPI_PASSWORD`
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestAuthenticate_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	_, err := authenticate(server.URL, "user", "pass")
	if err == nil {
		t.Fatal("expected error for forbidden response, got nil")
	}

	expectedErr := `BMC rejected credentials for "user" (status 403):  - check TURINGPI_USERNAME/TURINGPI_PASSWORD`
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestAuthenticate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := authenticate(server.URL, "user", "pass")
	if err == nil {
		t.Fatal("expected error for server error, got nil")
	}

	expectedErr := "BMC authentication returned status 500: "
	if err.Error() != expectedErr {
		t.Errorf("expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestAuthenticate_ConnectionError(t *testing.T) {
	// Use an invalid URL to simulate connection error
	_, err := authenticate("http://localhost:99999", "user", "pass")
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestAuthenticate_EmptyToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{})
	}))
	defer server.Close()

	token, err := authenticate(server.URL, "user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Empty token should be returned when "id" key is missing
	if token != "" {
		t.Errorf("expected empty token, got %s", token)
	}
}

func TestAuthenticate_EndpointURLConstruction(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantPath string
	}{
		{
			name:     "without trailing slash",
			endpoint: "https://turingpi.local",
			wantPath: "/api/bmc/authenticate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]string{"id": "token"})
			}))
			defer server.Close()

			_, _ = authenticate(server.URL, "user", "pass")

			if capturedPath != tt.wantPath {
				t.Errorf("expected path %s, got %s", tt.wantPath, capturedPath)
			}
		})
	}
}

package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func authenticate(endpoint, username, password string) (string, error) {
	url := fmt.Sprintf("%s/api/bmc/authenticate", endpoint)
	data := map[string]string{"username": username, "password": password}
	jsonData, _ := json.Marshal(data)

	resp, err := HTTPClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("connecting to BMC at %s: %w", endpoint, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return "", fmt.Errorf("BMC rejected credentials for %q (status %d): %s - check TURINGPI_USERNAME/TURINGPI_PASSWORD", username, resp.StatusCode, body)
		}
		return "", fmt.Errorf("BMC authentication returned status %d: %s", resp.StatusCode, body)
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode authentication response: %v", err)
	}
	return result["id"], nil
}

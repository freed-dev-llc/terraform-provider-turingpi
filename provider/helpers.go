package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Note: Uses HTTPClient from provider.go for TLS configuration

// checkPowerStatus returns "on" or "off" for the given 1-indexed node by
// querying the BMC power endpoint.
func checkPowerStatus(endpoint, token string, node int) (string, error) {
	status, err := getPowerStatus(endpoint, token)
	if err != nil {
		return "", fmt.Errorf("failed to read power status for node %d: %w", node, err)
	}
	if parsePowerStatus(status)[fmt.Sprintf("node%d", node)] {
		return "on", nil
	}
	return "off", nil
}

func checkBootStatus(ctx context.Context, endpoint string, node int, timeout int, token string, pattern string) (bool, error) {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=uart&node=%d", endpoint, node)

	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return false, err
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return false, fmt.Errorf("failed to create UART request: %v", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := HTTPClient.Do(req)
		if err != nil {
			return false, fmt.Errorf("UART request failed: %v", err)
		}

		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, fmt.Errorf("failed to read UART response: %v", err)
		}

		// Check for configured boot pattern in UART output
		if strings.Contains(string(body), pattern) {
			fmt.Printf("Node %d booted successfully: pattern %q detected.\n", node, pattern)
			return true, nil
		}

		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(5 * time.Second):
		}
	}

	return false, fmt.Errorf("timeout reached: node %d did not boot successfully (pattern %q not found)", node, pattern)
}

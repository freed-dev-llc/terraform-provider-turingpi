package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// powerStatusResponse represents the response from GET /api/bmc?opt=get&type=power
// Supports both legacy format ([][]interface{}) and new format ([]map[string]interface{})
type powerStatusResponse struct {
	Response json.RawMessage `json:"response"`
}

func dataSourcePower() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves the current power status of all nodes on the Turing Pi BMC.",
		ReadContext: dataSourcePowerRead,
		Schema: map[string]*schema.Schema{
			"node1": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Power status of node 1 (true = powered on, false = powered off)",
			},
			"node2": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Power status of node 2 (true = powered on, false = powered off)",
			},
			"node3": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Power status of node 3 (true = powered on, false = powered off)",
			},
			"node4": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Power status of node 4 (true = powered on, false = powered off)",
			},
			"nodes": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Power status of all nodes as a map (node1-node4 -> bool)",
				Elem: &schema.Schema{
					Type: schema.TypeBool,
				},
			},
			"powered_on_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of nodes currently powered on",
			},
			"powered_off_count": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of nodes currently powered off",
			},
		},
	}
}

func dataSourcePowerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	// Fetch power status
	status, err := getPowerStatus(config.Endpoint, config.Token)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read power status: %w", err))
	}

	// Parse the response
	nodeStatus := parsePowerStatus(status)

	// Set individual node values
	if err := d.Set("node1", nodeStatus["node1"]); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set node1: %w", err))
	}
	if err := d.Set("node2", nodeStatus["node2"]); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set node2: %w", err))
	}
	if err := d.Set("node3", nodeStatus["node3"]); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set node3: %w", err))
	}
	if err := d.Set("node4", nodeStatus["node4"]); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set node4: %w", err))
	}

	// Set nodes map
	if err := d.Set("nodes", nodeStatus); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set nodes: %w", err))
	}

	// Calculate counts
	poweredOn := 0
	poweredOff := 0
	for _, powered := range nodeStatus {
		if powered {
			poweredOn++
		} else {
			poweredOff++
		}
	}

	if err := d.Set("powered_on_count", poweredOn); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set powered_on_count: %w", err))
	}
	if err := d.Set("powered_off_count", poweredOff); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set powered_off_count: %w", err))
	}

	// Set a stable ID for the data source
	d.SetId("turingpi-power-status")

	return diags
}

// getPowerStatus fetches current power status from BMC
func getPowerStatus(endpoint, token string) (*powerStatusResponse, error) {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=power", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("power status request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("power status request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	var result powerStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// parsePowerStatus extracts node power status from API response
// Handles both legacy format and new BMC firmware format
func parsePowerStatus(status *powerStatusResponse) map[string]bool {
	nodes := make(map[string]bool)

	// Initialize all nodes to false
	nodes["node1"] = false
	nodes["node2"] = false
	nodes["node3"] = false
	nodes["node4"] = false

	// Try parsing as new format first: [{"result": [{"node1": "1", ...}]}]
	var newFormat []map[string]interface{}
	if err := json.Unmarshal(status.Response, &newFormat); err == nil {
		for _, item := range newFormat {
			// Check for "result" array in the response
			if result, ok := item["result"].([]interface{}); ok {
				for _, r := range result {
					if nodeMap, ok := r.(map[string]interface{}); ok {
						for nodeName, value := range nodeMap {
							nodes[nodeName] = parsePowerValue(value)
						}
					}
				}
			}
		}
		return nodes
	}

	// Fall back to legacy format: [[nodeName, status], [nodeName, status], ...]
	var legacyFormat [][]interface{}
	if err := json.Unmarshal(status.Response, &legacyFormat); err == nil {
		for _, item := range legacyFormat {
			if len(item) >= 2 {
				nodeName, nameOk := item[0].(string)
				if nameOk {
					nodes[nodeName] = parsePowerValue(item[1])
				}
			}
		}
	}

	return nodes
}

// parsePowerValue converts various types to boolean power state
func parsePowerValue(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val == 1
	case int:
		return val == 1
	case string:
		return val == "1" || val == "true" || val == "on"
	}
	return false
}

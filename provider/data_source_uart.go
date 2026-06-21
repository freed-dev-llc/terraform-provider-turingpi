package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// UART response structure
type uartReadResponse struct {
	Response [][]interface{} `json:"response"`
}

func dataSourceUART() *schema.Resource {
	return &schema.Resource{
		Description: "Reads buffered UART output from a Turing Pi compute node. Reading clears the UART buffer.",
		ReadContext: dataSourceUARTRead,
		Schema: map[string]*schema.Schema{
			"node": {
				Type:             schema.TypeInt,
				Required:         true,
				Description:      "Node ID to read UART output from (1-4)",
				ValidateDiagFunc: validateNodeID,
			},
			"encoding": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "utf8",
				Description:      "Character encoding for UART output. Valid values: utf8, utf16, utf16le, utf16be, utf32, utf32le, utf32be",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"utf8", "utf16", "utf16le", "utf16be", "utf32", "utf32le", "utf32be"}, false)),
			},
			// Computed attributes
			"output": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The buffered UART output from the node. Reading this clears the buffer.",
			},
			"has_output": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether there was any output in the UART buffer.",
			},
		},
	}
}

func dataSourceUARTRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	node := d.Get("node").(int)
	encoding := d.Get("encoding").(string)

	output, err := readUART(config.Endpoint, config.Token, node, encoding)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read UART: %w", err))
	}

	if err := d.Set("output", output); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set output: %w", err))
	}

	if err := d.Set("has_output", len(output) > 0); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set has_output: %w", err))
	}

	d.SetId(fmt.Sprintf("uart-node-%d", node))

	return diags
}

// readUART reads the buffered UART output from a node
func readUART(endpoint, token string, node int, encoding string) (string, error) {
	// API uses 0-indexed nodes
	apiNode := node - 1
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=uart&node=%d&encoding=%s", endpoint, apiNode, encoding)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result uartReadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return extractUARTOutput(result), nil
}

// extractUARTOutput extracts the UART output string from the response
func extractUARTOutput(resp uartReadResponse) string {
	for _, item := range resp.Response {
		if len(item) >= 2 {
			key, keyOk := item[0].(string)
			if keyOk && (key == "uart" || key == "output" || key == "data") {
				if value, valueOk := item[1].(string); valueOk {
					return value
				}
			}
		}
	}
	// If no structured response, try to get any string value
	for _, item := range resp.Response {
		if len(item) >= 1 {
			if value, valueOk := item[0].(string); valueOk {
				return value
			}
		}
	}
	return ""
}

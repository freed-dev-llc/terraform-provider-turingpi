package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceUART() *schema.Resource {
	return &schema.Resource{
		Description:   "Writes data to the UART of a Turing Pi compute node. Use this to send commands or data to a node's serial console.",
		CreateContext: resourceUARTCreate,
		ReadContext:   resourceUARTRead,
		UpdateContext: resourceUARTUpdate,
		DeleteContext: resourceUARTDelete,
		Schema: map[string]*schema.Schema{
			"node": {
				Type:             schema.TypeInt,
				Required:         true,
				Description:      "Node ID to write UART data to (1-4)",
				ValidateDiagFunc: validateNodeID,
			},
			"command": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The command or data to write to the UART. Will be URL-encoded automatically.",
			},
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will trigger resending the command.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed attributes
			"last_sent": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of when the command was last sent.",
			},
		},
	}
}

func resourceUARTCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	node := d.Get("node").(int)
	command := d.Get("command").(string)

	if err := writeUART(config.Endpoint, config.Token, node, command); err != nil {
		return diag.FromErr(fmt.Errorf("failed to write UART: %w", err))
	}

	d.SetId(fmt.Sprintf("uart-write-node-%d", node))
	if err := d.Set("last_sent", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_sent: %w", err))
	}

	return nil
}

func resourceUARTRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// UART write is a trigger resource - nothing to read back
	// The state is maintained locally
	return nil
}

func resourceUARTUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	// Resend command if it changed or triggers changed
	if d.HasChange("command") || d.HasChange("triggers") || d.HasChange("node") {
		node := d.Get("node").(int)
		command := d.Get("command").(string)

		if err := writeUART(config.Endpoint, config.Token, node, command); err != nil {
			return diag.FromErr(fmt.Errorf("failed to write UART: %w", err))
		}

		// Update ID if node changed
		d.SetId(fmt.Sprintf("uart-write-node-%d", node))
		if err := d.Set("last_sent", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_sent: %w", err))
		}
	}

	return nil
}

func resourceUARTDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Nothing to clean up - UART write is ephemeral
	d.SetId("")
	return nil
}

// writeUART sends a command to a node's UART
func writeUART(endpoint, token string, node int, command string) error {
	// API uses 0-indexed nodes
	apiNode := node - 1
	// URL-encode the command
	encodedCmd := url.QueryEscape(command)
	apiURL := fmt.Sprintf("%s/api/bmc?opt=set&type=uart&node=%d&cmd=%s", endpoint, apiNode, encodedCmd)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

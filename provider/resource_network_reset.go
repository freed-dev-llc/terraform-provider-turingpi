package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceNetworkReset() *schema.Resource {
	return &schema.Resource{
		Description:   "Triggers a reset of the Turing Pi network switch. The switch will be reset when this resource is created or when the triggers change.",
		CreateContext: resourceNetworkResetCreate,
		ReadContext:   resourceNetworkResetRead,
		UpdateContext: resourceNetworkResetUpdate,
		DeleteContext: resourceNetworkResetDelete,
		Schema: map[string]*schema.Schema{
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will trigger a network reset. Use this to force a reset based on other resource changes.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"last_reset": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of the last network reset operation.",
			},
		},
	}
}

func resourceNetworkResetCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	if err := resetNetwork(config.Endpoint, config.Token); err != nil {
		return diag.FromErr(fmt.Errorf("failed to reset network: %w", err))
	}

	d.SetId("network-reset")
	if err := d.Set("last_reset", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_reset: %w", err))
	}

	return nil
}

func resourceNetworkResetRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Network reset is a trigger resource - nothing to read from API
	// Just preserve the current state
	return nil
}

func resourceNetworkResetUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	// If triggers changed, perform a reset
	if d.HasChange("triggers") {
		if err := resetNetwork(config.Endpoint, config.Token); err != nil {
			return diag.FromErr(fmt.Errorf("failed to reset network: %w", err))
		}

		if err := d.Set("last_reset", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_reset: %w", err))
		}
	}

	return nil
}

func resourceNetworkResetDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Nothing to clean up for a trigger resource
	d.SetId("")
	return nil
}

// resetNetwork triggers a network switch reset
func resetNetwork(endpoint, token string) error {
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=network", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("network switch reset request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("network switch reset request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

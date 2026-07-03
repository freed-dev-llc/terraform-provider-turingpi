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

func resourceUSBBoot() *schema.Resource {
	return &schema.Resource{
		Description:   "Enables USB boot mode for a specified node. For Raspberry Pi CM4s, this pulls pin 93 (nRPIBOOT) low to enable USB boot.",
		CreateContext: resourceUSBBootCreate,
		ReadContext:   resourceUSBBootRead,
		UpdateContext: resourceUSBBootUpdate,
		DeleteContext: resourceUSBBootDelete,
		Schema: map[string]*schema.Schema{
			"node": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validateNodeID,
				Description:      "The node number (1-4) to enable USB boot mode for.",
			},
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will re-enable USB boot mode.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed attributes
			"last_enabled": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when USB boot mode was last enabled.",
			},
		},
	}
}

func resourceUSBBootCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	if err := enableUSBBoot(config.Endpoint, config.Token, node); err != nil {
		return diag.FromErr(fmt.Errorf("failed to enable USB boot for node %d: %w", node, err))
	}

	d.SetId(fmt.Sprintf("usb-boot-node-%d", node))
	if err := d.Set("last_enabled", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_enabled: %w", err))
	}

	return nil
}

func resourceUSBBootRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// USB boot is a trigger resource - nothing to read back
	return nil
}

func resourceUSBBootUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	// Re-enable if node or triggers changed
	if d.HasChange("node") || d.HasChange("triggers") {
		if err := enableUSBBoot(config.Endpoint, config.Token, node); err != nil {
			return diag.FromErr(fmt.Errorf("failed to enable USB boot for node %d: %w", node, err))
		}

		if err := d.Set("last_enabled", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_enabled: %w", err))
		}
	}

	return nil
}

func resourceUSBBootDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Clear USB boot on delete
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	if err := clearUSBBoot(config.Endpoint, config.Token, node); err != nil {
		return diag.FromErr(fmt.Errorf("failed to clear USB boot for node %d: %w", node, err))
	}

	d.SetId("")
	return nil
}

// enableUSBBoot enables USB boot mode for a node
func enableUSBBoot(endpoint, token string, node int) error {
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=usb_boot&node=%d", endpoint, node)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("USB boot enable request for node %d failed: %w", node, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("USB boot enable request for node %d returned status %d: %s%s", node, resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

// clearUSBBoot clears USB boot status for a node
func clearUSBBoot(endpoint, token string, node int) error {
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=clear_usb_boot&node=%d", endpoint, node)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("USB boot clear request for node %d failed: %w", node, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("USB boot clear request for node %d returned status %d: %s%s", node, resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

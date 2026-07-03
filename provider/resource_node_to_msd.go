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

func resourceNodeToMSD() *schema.Resource {
	return &schema.Resource{
		Description:   "Reboots a node into USB Mass Storage Device (MSD) mode. This allows the node's storage to be accessed as a USB device for imaging or file transfer.",
		CreateContext: resourceNodeToMSDCreate,
		ReadContext:   resourceNodeToMSDRead,
		UpdateContext: resourceNodeToMSDUpdate,
		DeleteContext: resourceNodeToMSDDelete,
		Schema: map[string]*schema.Schema{
			"node": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validateNodeID,
				Description:      "The node number (1-4) to reboot into MSD mode.",
			},
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will trigger a reboot into MSD mode.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed attributes
			"last_triggered": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when MSD mode was last triggered.",
			},
		},
	}
}

func resourceNodeToMSDCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	if err := nodeToMSD(config.Endpoint, config.Token, node); err != nil {
		return diag.FromErr(fmt.Errorf("failed to reboot node %d into MSD mode: %w", node, err))
	}

	d.SetId(fmt.Sprintf("node-to-msd-%d", node))
	if err := d.Set("last_triggered", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_triggered: %w", err))
	}

	return nil
}

func resourceNodeToMSDRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// MSD mode is a trigger resource - nothing to read back
	return nil
}

func resourceNodeToMSDUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	// Re-trigger if node or triggers changed
	if d.HasChange("node") || d.HasChange("triggers") {
		if err := nodeToMSD(config.Endpoint, config.Token, node); err != nil {
			return diag.FromErr(fmt.Errorf("failed to reboot node %d into MSD mode: %w", node, err))
		}

		if err := d.Set("last_triggered", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_triggered: %w", err))
		}
	}

	return nil
}

func resourceNodeToMSDDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Nothing to clean up for a trigger resource
	d.SetId("")
	return nil
}

// nodeToMSD reboots a node into USB Mass Storage Device mode
func nodeToMSD(endpoint, token string, node int) error {
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=node_to_msd&node=%d", endpoint, node)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("node %d MSD-mode request failed: %w", node, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node %d MSD-mode request returned status %d: %s%s", node, resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceClearUSBBoot() *schema.Resource {
	return &schema.Resource{
		Description:   "Clears USB boot status for a specified node. This resets the USB boot mode configuration that was previously enabled.",
		CreateContext: resourceClearUSBBootCreate,
		ReadContext:   resourceClearUSBBootRead,
		UpdateContext: resourceClearUSBBootUpdate,
		DeleteContext: resourceClearUSBBootDelete,
		Schema: map[string]*schema.Schema{
			"node": {
				Type:             schema.TypeInt,
				Required:         true,
				ValidateDiagFunc: validateNodeID,
				Description:      "The node number (1-4) to clear USB boot status for.",
			},
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will re-clear USB boot status.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			// Computed attributes
			"last_cleared": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp when USB boot status was last cleared.",
			},
		},
	}
}

func resourceClearUSBBootCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	if err := clearUSBBoot(config.Endpoint, config.Token, node); err != nil {
		return diag.FromErr(fmt.Errorf("failed to clear USB boot for node %d: %w", node, err))
	}

	d.SetId(fmt.Sprintf("clear-usb-boot-node-%d", node))
	if err := d.Set("last_cleared", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_cleared: %w", err))
	}

	return nil
}

func resourceClearUSBBootRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Clear USB boot is a trigger resource - nothing to read back
	return nil
}

func resourceClearUSBBootUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	// Re-clear if node or triggers changed
	if d.HasChange("node") || d.HasChange("triggers") {
		if err := clearUSBBoot(config.Endpoint, config.Token, node); err != nil {
			return diag.FromErr(fmt.Errorf("failed to clear USB boot for node %d: %w", node, err))
		}

		if err := d.Set("last_cleared", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_cleared: %w", err))
		}
	}

	return nil
}

func resourceClearUSBBootDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Nothing to clean up for a trigger resource
	d.SetId("")
	return nil
}

package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceNode() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNodeProvision,
		ReadContext:   resourceNodeStatus,
		UpdateContext: resourceNodeProvision,
		DeleteContext: resourceNodeDelete,
		Schema: map[string]*schema.Schema{
			"node": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Node ID to manage",
			},
			"firmware_file": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to the firmware file (required for flashing)",
			},
			"power_state": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "on",
				Description: "Desired power state of the node (on/off)",
			},
			"boot_check": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Check if the node successfully boots by monitoring UART output",
			},
			"login_prompt_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     60,
				Description: "Timeout in seconds to wait for boot check pattern via UART",
			},
			"boot_check_pattern": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "login:",
				Description: "Pattern to search for in UART output to confirm successful boot (e.g., 'login:' for standard Linux, 'machine is running and ready' for Talos)",
			},
		},
	}
}

func resourceNodeProvision(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)
	firmware := d.Get("firmware_file").(string)
	powerState := d.Get("power_state").(string)
	bootCheck := d.Get("boot_check").(bool)
	timeout := d.Get("login_prompt_timeout").(int)
	bootCheckPattern := d.Get("boot_check_pattern").(string)

	// Step 1: Turn on the node
	if powerState == "on" {
		turnOnNode(node)
	} else {
		turnOffNode(node)
	}

	// Step 2: Flash firmware if provided
	if firmware != "" {
		flashNode(node, firmware)
	}

	// Step 3: Boot check
	if bootCheck {
		fmt.Printf("Checking boot status for node %d (pattern: %q)...\n", node, bootCheckPattern)
		success, err := checkBootStatus(config.Endpoint, node, timeout, config.Token, bootCheckPattern)
		if err != nil {
			return diag.FromErr(fmt.Errorf("boot status check failed for node %d: %v", node, err))
		}
		if !success {
			return diag.Errorf("node %d did not boot successfully", node)
		}
	}

	d.SetId(fmt.Sprintf("node-%d", node))
	return nil
}

func resourceNodeStatus(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	node := d.Get("node").(int)
	currentPower := checkPowerStatus(node)

	if err := d.Set("power_state", currentPower); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set power_state: %v", err))
	}
	return nil
}

func resourceNodeDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	node := d.Get("node").(int)
	turnOffNode(node)
	return nil
}

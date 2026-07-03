package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourcePower() *schema.Resource {
	return &schema.Resource{
		Description:   "Manages power state of a Turing Pi compute node, including power on, power off, and reset operations.",
		CreateContext: resourcePowerCreate,
		ReadContext:   resourcePowerRead,
		UpdateContext: resourcePowerUpdate,
		DeleteContext: resourcePowerDelete,
		Schema: map[string]*schema.Schema{
			"node": {
				Type:             schema.TypeInt,
				Required:         true,
				Description:      "Node ID to control power (1-4)",
				ValidateDiagFunc: validateNodeID,
			},
			"state": {
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Power state: 'on', 'off', or 'reset'. Reset triggers a reboot and the state returns to 'on' after.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"on", "off", "reset"}, false)),
			},
			// Computed attribute showing actual power state
			"current_state": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Current power state as reported by BMC (true = powered on)",
			},
		},
		Importer: &schema.ResourceImporter{
			StateContext: resourcePowerImport,
		},
	}
}

func resourcePowerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	node := d.Get("node").(int)
	state := d.Get("state").(string)

	if err := setPowerState(config.Endpoint, config.Token, node, state); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set power state: %w", err))
	}

	d.SetId(fmt.Sprintf("power-node-%d", node))

	// Read back the state
	return resourcePowerRead(ctx, d, meta)
}

func resourcePowerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	node := d.Get("node").(int)

	// Fetch power status
	status, err := getPowerStatus(config.Endpoint, config.Token)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to read power status: %w", err))
	}

	nodeStatus := parsePowerStatus(status)
	nodeName := fmt.Sprintf("node%d", node)

	powered, ok := nodeStatus[nodeName]
	if !ok {
		// Node not found in response, assume off
		powered = false
	}

	if err := d.Set("current_state", powered); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set current_state: %w", err))
	}

	return diags
}

func resourcePowerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	node := d.Get("node").(int)
	state := d.Get("state").(string)

	if err := setPowerState(config.Endpoint, config.Token, node, state); err != nil {
		return diag.FromErr(fmt.Errorf("failed to update power state: %w", err))
	}

	// Update ID if node changed
	d.SetId(fmt.Sprintf("power-node-%d", node))

	// Read back the state
	return resourcePowerRead(ctx, d, meta)
}

func resourcePowerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	node := d.Get("node").(int)

	// On delete, power off the node
	if err := setPowerState(config.Endpoint, config.Token, node, "off"); err != nil {
		return diag.FromErr(fmt.Errorf("failed to power off node on delete: %w", err))
	}

	d.SetId("")
	return nil
}

func resourcePowerImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Import format: node ID (1-4)
	// The ID will be set by the import command
	id := d.Id()

	var node int
	if _, err := fmt.Sscanf(id, "%d", &node); err != nil {
		return nil, fmt.Errorf("invalid import ID '%s': expected node number (1-4)", id)
	}

	if node < 1 || node > 4 {
		return nil, fmt.Errorf("invalid node number %d: must be between 1 and 4", node)
	}

	if err := d.Set("node", node); err != nil {
		return nil, fmt.Errorf("failed to set node: %w", err)
	}

	// Set default state to "on" - will be updated by Read
	if err := d.Set("state", "on"); err != nil {
		return nil, fmt.Errorf("failed to set state: %w", err)
	}

	d.SetId(fmt.Sprintf("power-node-%d", node))

	return []*schema.ResourceData{d}, nil
}

// setPowerState sets the power state for a node
func setPowerState(endpoint, token string, node int, state string) error {
	switch state {
	case "on":
		return setNodePower(endpoint, token, node, true)
	case "off":
		return setNodePower(endpoint, token, node, false)
	case "reset":
		return resetNode(endpoint, token, node)
	default:
		return fmt.Errorf("invalid state: %s", state)
	}
}

// setNodePower turns a node on or off
func setNodePower(endpoint, token string, node int, powerOn bool) error {
	// API uses node1, node2, etc. parameters
	powerValue := "0"
	if powerOn {
		powerValue = "1"
	}

	url := fmt.Sprintf("%s/api/bmc?opt=set&type=power&node%d=%s", endpoint, node, powerValue)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("power request for node %d failed: %w", node, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("power request for node %d returned status %d: %s%s", node, resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

// resetNode triggers a reset/reboot of the specified node
func resetNode(endpoint, token string, node int) error {
	// API uses 0-indexed nodes for reset
	apiNode := node - 1
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=reset&node=%d", endpoint, apiNode)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("reset request for node %d failed: %w", node, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reset request for node %d returned status %d: %s%s", node, resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

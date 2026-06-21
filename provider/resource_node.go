package provider

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceNode() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceNodeCreate,
		ReadContext:   resourceNodeStatus,
		UpdateContext: resourceNodeUpdate,
		DeleteContext: resourceNodeDelete,
		// Flashing can run on both create and update and may take many minutes;
		// give it the same headroom as turingpi_flash instead of the SDK's 20m
		// default, and let users override via a timeouts {} block.
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(30 * time.Minute),
			Update: schema.DefaultTimeout(30 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"node": {
				Type:             schema.TypeInt,
				Required:         true,
				ForceNew:         true,
				Description:      "Node ID to manage (1-4)",
				ValidateDiagFunc: validateNodeID,
			},
			"firmware_url": {
				Type:             schema.TypeString,
				Optional:         true,
				Description:      "HTTP(S) URL the BMC pulls the firmware image from. This is the reliable flash path: the BMC reports completion only after the eMMC write finishes. Mutually exclusive with firmware_file.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringMatch(regexp.MustCompile(`^https?://`), "must be an http(s) URL")),
				ConflictsWith:    []string{"firmware_file"},
			},
			"firmware_file": {
				Type:          schema.TypeString,
				Optional:      true,
				Deprecated:    "Use firmware_url instead. firmware_file uses the BMC streaming-upload path, which is known broken on current firmware (issue #63): the BMC reports completion before the eMMC write finishes, so the apply can succeed against a node that was never actually flashed. firmware_url has the BMC pull the image itself and report accurate completion.",
				Description:   "Path to a local firmware image streamed to the node. Deprecated: prefer firmware_url. Mutually exclusive with firmware_url.",
				ConflictsWith: []string{"firmware_url"},
			},
			"power_state": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          "on",
				Description:      "Desired power state of the node (on/off)",
				ValidateDiagFunc: validation.ToDiagFunc(validation.StringInSlice([]string{"on", "off"}, false)),
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

// firmwareConfigured reports whether either flash input is set.
func firmwareConfigured(d *schema.ResourceData) bool {
	return d.Get("firmware_url").(string) != "" || d.Get("firmware_file").(string) != ""
}

func resourceNodeCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// On create, flash whenever a firmware source is configured.
	return applyNode(ctx, d, meta.(*ProviderConfig), firmwareConfigured(d), true)
}

func resourceNodeUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// On update, only re-flash when a firmware source actually changed.
	// Otherwise an unrelated edit (e.g. toggling power_state) would re-image the
	// eMMC on every apply.
	doFlash := firmwareConfigured(d) && (d.HasChange("firmware_url") || d.HasChange("firmware_file"))
	return applyNode(ctx, d, meta.(*ProviderConfig), doFlash, false)
}

// applyNode runs the provision sequence: optionally flash, apply the desired
// power state, then optionally wait for boot. doFlash decides whether a flash
// runs this round; isCreate is true on Create (power is always converged) and
// false on Update (power is applied only when it changed).
func applyNode(ctx context.Context, d *schema.ResourceData, config *ProviderConfig, doFlash bool, isCreate bool) diag.Diagnostics {
	node := d.Get("node").(int)
	powerState := d.Get("power_state").(string)
	bootCheck := d.Get("boot_check").(bool)
	timeout := d.Get("login_prompt_timeout").(int)
	bootCheckPattern := d.Get("boot_check_pattern").(string)

	// Step 1: Flash firmware. Flashing powers the node off, so it must run
	// before we apply the desired power state below. Prefer the reliable URL
	// path; fall back to the deprecated streaming file path.
	if doFlash {
		apiNode := node - 1 // BMC flash API is 0-indexed
		if firmwareURL := d.Get("firmware_url").(string); firmwareURL != "" {
			if err := flashFirmwareURL(ctx, config.Endpoint, config.Token, node, apiNode, firmwareURL); err != nil {
				return diag.FromErr(fmt.Errorf("failed to flash node %d from URL: %w", node, err))
			}
		} else {
			firmwareFile := d.Get("firmware_file").(string)
			tflog.Warn(ctx, "turingpi_node: firmware_file uses the streaming upload flash path, which is known broken on current BMC firmware - the BMC reports Done before the eMMC write completes (issue #63), so boot_check may pass against the previously installed OS. Use firmware_url for reliable flashing.")
			if err := flashFirmwareFile(ctx, config.Endpoint, config.Token, node, apiNode, firmwareFile); err != nil {
				return diag.FromErr(fmt.Errorf("failed to flash node %d: %w", node, err))
			}
		}
	}

	// Step 2: Apply the desired power state, reusing the same dispatch as
	// turingpi_power. Skip the call when it would be a no-op: on update when
	// power_state did not change, and right after a flash that already left the
	// node powered off.
	needPower := isCreate || d.HasChange("power_state") || doFlash
	if doFlash && powerState == "off" {
		needPower = false // flashing already left the node powered off
	}
	if needPower {
		if err := setPowerState(config.Endpoint, config.Token, node, powerState); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set power state for node %d: %w", node, err))
		}
	}

	// Record the resource now that the node exists and its power state is
	// applied. Setting the ID before the optional boot check means a boot-check
	// failure leaves a tracked (tainted) resource rather than an untracked node
	// that gets re-flashed from scratch on the next apply.
	d.SetId(fmt.Sprintf("node-%d", node))

	// Step 3: Boot check.
	if bootCheck {
		fmt.Printf("Checking boot status for node %d (pattern: %q)...\n", node, bootCheckPattern)
		success, err := checkBootStatus(ctx, config.Endpoint, node, timeout, config.Token, bootCheckPattern)
		if err != nil {
			return diag.FromErr(fmt.Errorf("boot status check failed for node %d: %v", node, err))
		}
		if !success {
			return diag.Errorf("node %d did not boot successfully", node)
		}
	}

	return nil
}

func resourceNodeStatus(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)

	currentPower, err := checkPowerStatus(config.Endpoint, config.Token, node)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("power_state", currentPower); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set power_state: %v", err))
	}
	return nil
}

func resourceNodeDelete(_ context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	node := d.Get("node").(int)
	if err := setPowerState(config.Endpoint, config.Token, node, "off"); err != nil {
		return diag.FromErr(fmt.Errorf("failed to power off node %d on delete: %w", node, err))
	}
	return nil
}

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

func resourceBMCReboot() *schema.Resource {
	return &schema.Resource{
		Description:   "Triggers a reboot of the Turing Pi BMC (Baseboard Management Controller). The BMC will be temporarily unavailable during the reboot.",
		CreateContext: resourceBMCRebootCreate,
		ReadContext:   resourceBMCRebootRead,
		UpdateContext: resourceBMCRebootUpdate,
		DeleteContext: resourceBMCRebootDelete,
		Schema: map[string]*schema.Schema{
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will trigger a BMC reboot.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"wait_for_ready": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Wait for the BMC to become available again after reboot (default: true).",
			},
			"ready_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     120,
				Description: "Timeout in seconds to wait for BMC to become ready after reboot (default: 120).",
			},
			// Computed attributes
			"last_reboot": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of the last BMC reboot operation.",
			},
		},
	}
}

func resourceBMCRebootCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	waitForReady := d.Get("wait_for_ready").(bool)
	readyTimeout := d.Get("ready_timeout").(int)

	if err := rebootBMC(config.Endpoint, config.Token); err != nil {
		return diag.FromErr(fmt.Errorf("failed to reboot BMC: %w", err))
	}

	if waitForReady {
		if err := waitForBMCReady(config.Endpoint, config.Token, readyTimeout); err != nil {
			return diag.FromErr(fmt.Errorf("BMC did not become ready after reboot: %w", err))
		}
	}

	d.SetId("bmc-reboot")
	if err := d.Set("last_reboot", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_reboot: %w", err))
	}

	return nil
}

func resourceBMCRebootRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// BMC reboot is a trigger resource - nothing to read back
	return nil
}

func resourceBMCRebootUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	// Reboot if triggers changed
	if d.HasChange("triggers") {
		waitForReady := d.Get("wait_for_ready").(bool)
		readyTimeout := d.Get("ready_timeout").(int)

		if err := rebootBMC(config.Endpoint, config.Token); err != nil {
			return diag.FromErr(fmt.Errorf("failed to reboot BMC: %w", err))
		}

		if waitForReady {
			if err := waitForBMCReady(config.Endpoint, config.Token, readyTimeout); err != nil {
				return diag.FromErr(fmt.Errorf("BMC did not become ready after reboot: %w", err))
			}
		}

		if err := d.Set("last_reboot", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_reboot: %w", err))
		}
	}

	return nil
}

func resourceBMCRebootDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Nothing to clean up for a trigger resource
	d.SetId("")
	return nil
}

// rebootBMC triggers a BMC reboot
func rebootBMC(endpoint, token string) error {
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=reboot", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("BMC reboot request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("BMC reboot request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

// waitForBMCReady waits for the BMC to become available after reboot
func waitForBMCReady(endpoint, token string, timeoutSeconds int) error {
	// Wait a few seconds for the reboot to initiate
	time.Sleep(5 * time.Second)

	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	for time.Now().Before(deadline) {
		if checkBMCReady(endpoint, token) {
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout after %d seconds", timeoutSeconds)
}

// checkBMCReady checks if the BMC is responding to API requests
func checkBMCReady(endpoint, token string) bool {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=about", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("Authorization", "Bearer "+token)

	// Use a short timeout for health checks
	client := &http.Client{
		Timeout:   5 * time.Second,
		Transport: HTTPClient.Transport,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == http.StatusOK
}

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

func resourceBMCReload() *schema.Resource {
	return &schema.Resource{
		Description:   "Restarts the BMC system management daemon (bmcd). This is a softer restart than a full BMC reboot.",
		CreateContext: resourceBMCReloadCreate,
		ReadContext:   resourceBMCReloadRead,
		UpdateContext: resourceBMCReloadUpdate,
		DeleteContext: resourceBMCReloadDelete,
		Schema: map[string]*schema.Schema{
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will trigger a daemon reload.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"wait_for_ready": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Wait for the BMC daemon to become available again after reload (default: true).",
			},
			"ready_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     30,
				Description: "Timeout in seconds to wait for daemon to become ready after reload (default: 30).",
			},
			// Computed attributes
			"last_reload": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of the last daemon reload operation.",
			},
		},
	}
}

func resourceBMCReloadCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	waitForReady := d.Get("wait_for_ready").(bool)
	readyTimeout := d.Get("ready_timeout").(int)

	if err := reloadBMCDaemon(config.Endpoint, config.Token); err != nil {
		return diag.FromErr(fmt.Errorf("failed to reload BMC daemon: %w", err))
	}

	if waitForReady {
		if err := waitForBMCReady(config.Endpoint, config.Token, readyTimeout); err != nil {
			return diag.FromErr(fmt.Errorf("BMC daemon did not become ready after reload: %w", err))
		}
	}

	d.SetId("bmc-reload")
	if err := d.Set("last_reload", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_reload: %w", err))
	}

	return nil
}

func resourceBMCReloadRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// BMC reload is a trigger resource - nothing to read back
	return nil
}

func resourceBMCReloadUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	// Reload if triggers changed
	if d.HasChange("triggers") {
		waitForReady := d.Get("wait_for_ready").(bool)
		readyTimeout := d.Get("ready_timeout").(int)

		if err := reloadBMCDaemon(config.Endpoint, config.Token); err != nil {
			return diag.FromErr(fmt.Errorf("failed to reload BMC daemon: %w", err))
		}

		if waitForReady {
			if err := waitForBMCReady(config.Endpoint, config.Token, readyTimeout); err != nil {
				return diag.FromErr(fmt.Errorf("BMC daemon did not become ready after reload: %w", err))
			}
		}

		if err := d.Set("last_reload", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_reload: %w", err))
		}
	}

	return nil
}

func resourceBMCReloadDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Nothing to clean up for a trigger resource
	d.SetId("")
	return nil
}

// reloadBMCDaemon triggers a daemon reload
func reloadBMCDaemon(endpoint, token string) error {
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=reload", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("BMC daemon reload request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("BMC daemon reload request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	return nil
}

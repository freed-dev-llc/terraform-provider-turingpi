package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// sdcardResponse represents the BMC sdcard API response
type sdcardResponse struct {
	Response []sdcardInfo `json:"response"`
}

type sdcardInfo struct {
	Total int64 `json:"total"`
	Free  int64 `json:"free"`
	Use   int64 `json:"use"`
}

func dataSourceSDCard() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about the microSD card in the Turing Pi BMC, including total capacity, used space, and free space.",
		ReadContext: dataSourceSDCardRead,
		Schema: map[string]*schema.Schema{
			"total_bytes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Total microSD card capacity in bytes.",
			},
			"used_bytes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Used space on the microSD card in bytes.",
			},
			"free_bytes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Free space on the microSD card in bytes.",
			},
			"total_gb": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Total microSD card capacity in gigabytes.",
			},
			"used_gb": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Used space on the microSD card in gigabytes.",
			},
			"free_gb": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Free space on the microSD card in gigabytes.",
			},
			"used_percent": {
				Type:        schema.TypeFloat,
				Computed:    true,
				Description: "Percentage of microSD card space used.",
			},
		},
	}
}

func dataSourceSDCardRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	sdcard, err := fetchSDCardInfo(config.Endpoint, config.Token)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch SD card info: %w", err))
	}

	if len(sdcard.Response) == 0 {
		return diag.FromErr(fmt.Errorf("no SD card information returned from API"))
	}

	info := sdcard.Response[0]

	if err := d.Set("total_bytes", info.Total); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set total_bytes: %w", err))
	}
	if err := d.Set("used_bytes", info.Use); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set used_bytes: %w", err))
	}
	if err := d.Set("free_bytes", info.Free); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set free_bytes: %w", err))
	}

	// Calculate GB values (bytes / 1024^3)
	const bytesPerGB = 1024 * 1024 * 1024
	totalGB := float64(info.Total) / bytesPerGB
	usedGB := float64(info.Use) / bytesPerGB
	freeGB := float64(info.Free) / bytesPerGB

	if err := d.Set("total_gb", totalGB); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set total_gb: %w", err))
	}
	if err := d.Set("used_gb", usedGB); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set used_gb: %w", err))
	}
	if err := d.Set("free_gb", freeGB); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set free_gb: %w", err))
	}

	// Calculate percentage used
	var usedPercent float64
	if info.Total > 0 {
		usedPercent = (float64(info.Use) / float64(info.Total)) * 100
	}
	if err := d.Set("used_percent", usedPercent); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set used_percent: %w", err))
	}

	d.SetId("turingpi-sdcard")

	return diags
}

func fetchSDCardInfo(endpoint, token string) (*sdcardResponse, error) {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=sdcard", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("SD card status request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("SD card status request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	var result sdcardResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// BMC firmware response structures
type firmwareInitResponse struct {
	Response [][]interface{} `json:"response"`
}

type flashProgressResponse struct {
	Response [][]interface{} `json:"response"`
}

func resourceBMCFirmware() *schema.Resource {
	return &schema.Resource{
		Description:   "Upgrades the BMC firmware on the Turing Pi. The BMC will reboot after a successful firmware update.",
		CreateContext: resourceBMCFirmwareCreate,
		ReadContext:   resourceBMCFirmwareRead,
		UpdateContext: resourceBMCFirmwareUpdate,
		DeleteContext: resourceBMCFirmwareDelete,
		Schema: map[string]*schema.Schema{
			"firmware_file": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Path to the BMC firmware file. Can be a local path on the Terraform host (will be uploaded) or a path on the BMC filesystem (use with bmc_local=true).",
			},
			"bmc_local": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, the firmware_file path refers to a file on the BMC's local filesystem. If false (default), the file will be uploaded from the Terraform host.",
			},
			"triggers": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: "A map of values that, when changed, will trigger a firmware upgrade. Use this to force an upgrade based on version changes.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     300,
				Description: "Timeout in seconds for the firmware upgrade operation (default: 300).",
			},
			// Computed attributes
			"last_upgrade": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Timestamp of the last firmware upgrade operation.",
			},
			"previous_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The firmware version before the upgrade.",
			},
		},
	}
}

func resourceBMCFirmwareCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	// Get current firmware version before upgrade
	aboutData, err := fetchBMCAbout(config.Endpoint, config.Token)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get current firmware version: %w", err))
	}

	previousVersion := extractFirmwareVersion(aboutData)
	if err := d.Set("previous_version", previousVersion); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set previous_version: %w", err))
	}

	// Perform the firmware upgrade
	if err := performFirmwareUpgrade(config, d); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("bmc-firmware")
	if err := d.Set("last_upgrade", time.Now().UTC().Format(time.RFC3339)); err != nil {
		return diag.FromErr(fmt.Errorf("failed to set last_upgrade: %w", err))
	}

	return nil
}

func resourceBMCFirmwareRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Firmware upgrade is a trigger resource - nothing to read back
	// The firmware version can be read via turingpi_info data source
	return nil
}

func resourceBMCFirmwareUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)

	// Check if we should trigger an upgrade
	if d.HasChange("firmware_file") || d.HasChange("triggers") || d.HasChange("bmc_local") {
		// Get current firmware version before upgrade
		aboutData, err := fetchBMCAbout(config.Endpoint, config.Token)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to get current firmware version: %w", err))
		}

		previousVersion := extractFirmwareVersion(aboutData)
		if err := d.Set("previous_version", previousVersion); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set previous_version: %w", err))
		}

		// Perform the firmware upgrade
		if err := performFirmwareUpgrade(config, d); err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set("last_upgrade", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return diag.FromErr(fmt.Errorf("failed to set last_upgrade: %w", err))
		}
	}

	return nil
}

func resourceBMCFirmwareDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Nothing to clean up for firmware - it's already flashed
	d.SetId("")
	return nil
}

func performFirmwareUpgrade(config *ProviderConfig, d *schema.ResourceData) error {
	firmwareFile := d.Get("firmware_file").(string)
	bmcLocal := d.Get("bmc_local").(bool)
	timeout := d.Get("timeout").(int)

	var handle string
	var err error

	if bmcLocal {
		// File is on BMC filesystem
		handle, err = initBMCLocalFirmwareUpgrade(config.Endpoint, config.Token, firmwareFile)
	} else {
		// File needs to be uploaded from Terraform host
		handle, err = uploadAndInitFirmwareUpgrade(config.Endpoint, config.Token, firmwareFile)
	}

	if err != nil {
		return fmt.Errorf("failed to initiate firmware upgrade: %w", err)
	}

	// Poll for completion
	if err := waitForFirmwareUpgrade(config.Endpoint, config.Token, handle, timeout); err != nil {
		return fmt.Errorf("firmware upgrade failed: %w", err)
	}

	return nil
}

// initBMCLocalFirmwareUpgrade initiates a firmware upgrade from a file on the BMC
func initBMCLocalFirmwareUpgrade(endpoint, token, filePath string) (string, error) {
	// For local files, we don't know the size, so we'll let the BMC handle it
	url := fmt.Sprintf("%s/api/bmc?opt=set&type=firmware&local&file=%s", endpoint, filePath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("firmware upgrade init request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("firmware upgrade init returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	var result firmwareInitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	// Extract handle from response
	handle := extractHandle(result)
	if handle == "" {
		return "", fmt.Errorf("no handle returned from firmware init")
	}

	return handle, nil
}

// uploadAndInitFirmwareUpgrade uploads a firmware file and initiates the upgrade
func uploadAndInitFirmwareUpgrade(endpoint, token, filePath string) (string, error) {
	// Open and get file size
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open firmware file: %w", err)
	}
	defer func() { _ = file.Close() }()

	stat, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to stat firmware file: %w", err)
	}

	fileSize := stat.Size()

	// Step 1: Initialize the firmware upload
	initURL := fmt.Sprintf("%s/api/bmc?opt=set&type=firmware&length=%d", endpoint, fileSize)

	initReq, err := http.NewRequest("GET", initURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create init request: %w", err)
	}
	initReq.Header.Set("Authorization", "Bearer "+token)

	initResp, err := HTTPClient.Do(initReq)
	if err != nil {
		return "", fmt.Errorf("firmware upload init request failed: %w", err)
	}
	defer func() { _ = initResp.Body.Close() }()

	if initResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(initResp.Body)
		return "", fmt.Errorf("firmware upload init returned status %d: %s%s", initResp.StatusCode, body, authHint(initResp.StatusCode))
	}

	var initResult firmwareInitResponse
	if err := json.NewDecoder(initResp.Body).Decode(&initResult); err != nil {
		return "", fmt.Errorf("failed to decode init response: %w", err)
	}

	handle := extractHandle(initResult)
	if handle == "" {
		return "", fmt.Errorf("no handle returned from firmware init")
	}

	// Step 2: Upload the firmware file
	if err := uploadFirmwareData(endpoint, token, handle, file, filePath); err != nil {
		// Try to cancel on error
		_ = cancelFirmwareUpload(endpoint, token, handle)
		return "", fmt.Errorf("failed to upload firmware: %w", err)
	}

	return handle, nil
}

// uploadFirmwareData uploads the firmware file data to the BMC
func uploadFirmwareData(endpoint, token, handle string, file *os.File, filePath string) error {
	// Reset file position
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	// Read file content
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := part.Write(fileContent); err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/api/bmc/upload/%s", endpoint, handle)

	req, err := http.NewRequest("POST", uploadURL, body)
	if err != nil {
		return fmt.Errorf("failed to create upload request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("firmware upload request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("firmware upload returned status %d: %s%s", resp.StatusCode, respBody, authHint(resp.StatusCode))
	}

	return nil
}

// cancelFirmwareUpload cancels an in-progress firmware upload
func cancelFirmwareUpload(endpoint, token, handle string) error {
	url := fmt.Sprintf("%s/api/bmc/upload/%s/cancel", endpoint, handle)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create cancel request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("firmware upload cancel request for handle %s failed: %w", handle, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// waitForFirmwareUpgrade polls for firmware upgrade completion
func waitForFirmwareUpgrade(endpoint, token, handle string, timeoutSeconds int) error {
	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)

	for time.Now().Before(deadline) {
		progress, err := getFlashProgress(endpoint, token)
		if err != nil {
			// BMC might be rebooting, wait and retry
			time.Sleep(5 * time.Second)
			continue
		}

		status := extractFlashStatus(progress)

		switch status {
		case "done", "complete", "success":
			return nil
		case "error", "failed":
			return fmt.Errorf("firmware upgrade failed")
		case "idle":
			// Flash completed, BMC is idle
			return nil
		}

		time.Sleep(3 * time.Second)
	}

	return fmt.Errorf("timeout waiting for firmware upgrade to complete")
}

// getFlashProgress retrieves the current flash progress
func getFlashProgress(endpoint, token string) (*flashProgressResponse, error) {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=flash", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("firmware upgrade progress request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("firmware upgrade progress request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	var result flashProgressResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// extractHandle extracts the upload handle from the init response
func extractHandle(resp firmwareInitResponse) string {
	for _, item := range resp.Response {
		if len(item) >= 2 {
			key, keyOk := item[0].(string)
			if keyOk && key == "handle" {
				if value, valueOk := item[1].(string); valueOk {
					return value
				}
			}
		}
	}
	return ""
}

// extractFlashStatus extracts the flash status from the progress response
func extractFlashStatus(resp *flashProgressResponse) string {
	for _, item := range resp.Response {
		if len(item) >= 2 {
			key, keyOk := item[0].(string)
			if keyOk && key == "status" {
				if value, valueOk := item[1].(string); valueOk {
					return value
				}
			}
		}
	}
	return ""
}

// extractFirmwareVersion extracts the firmware version from the about response
func extractFirmwareVersion(data *bmcAboutResponse) string {
	aboutMap := parseAboutResponse(data)
	if v, ok := aboutMap["firmware"]; ok {
		return v
	}
	// Also check for "version" which is the daemon version in new format
	if v, ok := aboutMap["version"]; ok {
		return v
	}
	return ""
}

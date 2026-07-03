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

// BMC API response structures
// Use json.RawMessage to support both legacy and new BMC firmware formats
type bmcAboutResponse struct {
	Response json.RawMessage `json:"response"`
}

type bmcInfoResponse struct {
	Response json.RawMessage `json:"response"`
}

type networkInterface struct {
	Device string `json:"device"`
	IP     string `json:"ip"`
	MAC    string `json:"mac"`
}

type storageDevice struct {
	Name       string `json:"name"`
	Total      int64  `json:"total"`
	TotalBytes int64  `json:"total_bytes"`
	Free       int64  `json:"free"`
	BytesFree  int64  `json:"bytes_free"`
	Used       int64  `json:"use"`
}

type bmcPowerResponse struct {
	Response json.RawMessage `json:"response"`
}

func dataSourceInfo() *schema.Resource {
	return &schema.Resource{
		Description: "Retrieves information about the Turing Pi BMC including version info, network configuration, storage metrics, and node power status.",
		ReadContext: dataSourceInfoRead,
		Schema: map[string]*schema.Schema{
			// Version information from /api/bmc?opt=get&type=about
			"api_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "BMC API version",
			},
			"daemon_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "BMC daemon version",
			},
			"buildroot_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Buildroot version",
			},
			"firmware_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "BMC firmware version",
			},
			"build_time": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "BMC build timestamp",
			},

			// Network information from /api/bmc?opt=get&type=info
			"network_interfaces": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of network interfaces on the BMC",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Network interface device name",
						},
						"ip": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "IP address",
						},
						"mac": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "MAC address",
						},
					},
				},
			},

			// Storage information from /api/bmc?opt=get&type=info
			"storage_devices": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of storage devices",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Storage device name (e.g., 'bmc', 'microSD')",
						},
						"total_bytes": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Total storage capacity in bytes",
						},
						"used_bytes": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Used storage in bytes",
						},
						"free_bytes": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Free storage in bytes",
						},
					},
				},
			},

			// Power status from /api/bmc?opt=get&type=power
			"nodes": {
				Type:        schema.TypeMap,
				Computed:    true,
				Description: "Power status of each node (node1-node4)",
				Elem: &schema.Schema{
					Type: schema.TypeBool,
				},
			},
		},
	}
}

func dataSourceInfoRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	config := meta.(*ProviderConfig)
	var diags diag.Diagnostics

	// Fetch version/about information
	aboutData, err := fetchBMCAbout(config.Endpoint, config.Token)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch BMC about info: %w", err))
	}

	if err := setAboutData(d, aboutData); err != nil {
		return diag.FromErr(err)
	}

	// Fetch network and storage information
	infoData, err := fetchBMCInfo(config.Endpoint, config.Token)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch BMC info: %w", err))
	}

	if err := setInfoData(d, infoData); err != nil {
		return diag.FromErr(err)
	}

	// Fetch power status
	powerData, err := fetchBMCPower(config.Endpoint, config.Token)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to fetch BMC power status: %w", err))
	}

	if err := setPowerData(d, powerData); err != nil {
		return diag.FromErr(err)
	}

	// Set a stable ID for the data source
	d.SetId("turingpi-bmc-info")

	return diags
}

func fetchBMCAbout(endpoint, token string) (*bmcAboutResponse, error) {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=about", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BMC \"about\" request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BMC \"about\" request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	var result bmcAboutResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func fetchBMCInfo(endpoint, token string) (*bmcInfoResponse, error) {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=info", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BMC info request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BMC info request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	var result bmcInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func fetchBMCPower(endpoint, token string) (*bmcPowerResponse, error) {
	url := fmt.Sprintf("%s/api/bmc?opt=get&type=power", endpoint)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BMC power status request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("BMC power status request returned status %d: %s%s", resp.StatusCode, body, authHint(resp.StatusCode))
	}

	var result bmcPowerResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func setAboutData(d *schema.ResourceData, data *bmcAboutResponse) error {
	aboutMap := parseAboutResponse(data)

	if v, ok := aboutMap["api"]; ok {
		if err := d.Set("api_version", v); err != nil {
			return fmt.Errorf("failed to set api_version: %w", err)
		}
	}
	if v, ok := aboutMap["version"]; ok {
		if err := d.Set("daemon_version", v); err != nil {
			return fmt.Errorf("failed to set daemon_version: %w", err)
		}
	}
	if v, ok := aboutMap["buildroot"]; ok {
		if err := d.Set("buildroot_version", v); err != nil {
			return fmt.Errorf("failed to set buildroot_version: %w", err)
		}
	}
	if v, ok := aboutMap["firmware"]; ok {
		if err := d.Set("firmware_version", v); err != nil {
			return fmt.Errorf("failed to set firmware_version: %w", err)
		}
	}
	// Handle both "buildtime" (legacy) and "build_version" (new) field names
	if v, ok := aboutMap["buildtime"]; ok {
		if err := d.Set("build_time", v); err != nil {
			return fmt.Errorf("failed to set build_time: %w", err)
		}
	} else if v, ok := aboutMap["build_version"]; ok {
		if err := d.Set("build_time", v); err != nil {
			return fmt.Errorf("failed to set build_time: %w", err)
		}
	}

	return nil
}

// parseAboutResponse extracts about data from API response
// Handles both legacy format and new BMC firmware format (2.3.4+)
func parseAboutResponse(data *bmcAboutResponse) map[string]string {
	aboutMap := make(map[string]string)

	// Try parsing as new format first: [{"result": {key: value, ...}}]
	var newFormat []map[string]interface{}
	if err := json.Unmarshal(data.Response, &newFormat); err == nil {
		for _, item := range newFormat {
			if result, ok := item["result"].(map[string]interface{}); ok {
				for key, value := range result {
					if strVal, ok := value.(string); ok {
						aboutMap[key] = strVal
					}
				}
			}
		}
		if len(aboutMap) > 0 {
			return aboutMap
		}
	}

	// Fall back to legacy format: [[key, value], [key, value], ...]
	var legacyFormat [][]interface{}
	if err := json.Unmarshal(data.Response, &legacyFormat); err == nil {
		for _, item := range legacyFormat {
			if len(item) >= 2 {
				key, keyOk := item[0].(string)
				value, valueOk := item[1].(string)
				if keyOk && valueOk {
					aboutMap[key] = value
				}
			}
		}
	}

	return aboutMap
}

func setInfoData(d *schema.ResourceData, data *bmcInfoResponse) error {
	networks, storages := parseInfoResponse(data)

	// Set network interfaces
	networkInterfaces := make([]map[string]interface{}, 0, len(networks))
	for _, iface := range networks {
		networkInterfaces = append(networkInterfaces, map[string]interface{}{
			"device": iface.Device,
			"ip":     iface.IP,
			"mac":    iface.MAC,
		})
	}
	if err := d.Set("network_interfaces", networkInterfaces); err != nil {
		return fmt.Errorf("failed to set network_interfaces: %w", err)
	}

	// Set storage devices
	storageDevices := make([]map[string]interface{}, 0, len(storages))
	for _, storage := range storages {
		// Handle both legacy (Total/Free) and new (TotalBytes/BytesFree) field names
		totalBytes := storage.Total
		if totalBytes == 0 {
			totalBytes = storage.TotalBytes
		}
		freeBytes := storage.Free
		if freeBytes == 0 {
			freeBytes = storage.BytesFree
		}
		storageDevices = append(storageDevices, map[string]interface{}{
			"name":        storage.Name,
			"total_bytes": totalBytes,
			"used_bytes":  storage.Used,
			"free_bytes":  freeBytes,
		})
	}
	if err := d.Set("storage_devices", storageDevices); err != nil {
		return fmt.Errorf("failed to set storage_devices: %w", err)
	}

	return nil
}

// parseInfoResponse extracts network and storage data from API response
// Handles both legacy format and new BMC firmware format (2.3.4+)
func parseInfoResponse(data *bmcInfoResponse) ([]networkInterface, []storageDevice) {
	var networks []networkInterface
	var storages []storageDevice

	// Try parsing as new format first: [{"result": {"ip": [...], "storage": [...]}}]
	var newFormat []map[string]interface{}
	if err := json.Unmarshal(data.Response, &newFormat); err == nil {
		for _, item := range newFormat {
			if result, ok := item["result"].(map[string]interface{}); ok {
				// Parse IP/network interfaces
				if ipList, ok := result["ip"].([]interface{}); ok {
					for _, ip := range ipList {
						if ipMap, ok := ip.(map[string]interface{}); ok {
							iface := networkInterface{
								Device: getStringValue(ipMap, "device"),
								IP:     getStringValue(ipMap, "ip"),
								MAC:    getStringValue(ipMap, "mac"),
							}
							networks = append(networks, iface)
						}
					}
				}
				// Parse storage devices
				if storageList, ok := result["storage"].([]interface{}); ok {
					for _, s := range storageList {
						if storageMap, ok := s.(map[string]interface{}); ok {
							storage := storageDevice{
								Name:       getStringValue(storageMap, "name"),
								TotalBytes: getInt64Value(storageMap, "total_bytes"),
								BytesFree:  getInt64Value(storageMap, "bytes_free"),
							}
							storages = append(storages, storage)
						}
					}
				}
			}
		}
		if len(networks) > 0 || len(storages) > 0 {
			return networks, storages
		}
	}

	// Fall back to legacy format: {"network": [...], "storage": [...]}
	var legacyFormat struct {
		Network []networkInterface `json:"network"`
		Storage []storageDevice    `json:"storage"`
	}
	if err := json.Unmarshal(data.Response, &legacyFormat); err == nil {
		networks = legacyFormat.Network
		storages = legacyFormat.Storage
	}

	return networks, storages
}

// getStringValue safely extracts a string value from a map
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getInt64Value safely extracts an int64 value from a map
func getInt64Value(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	if v, ok := m[key].(int64); ok {
		return v
	}
	if v, ok := m[key].(int); ok {
		return int64(v)
	}
	return 0
}

func setPowerData(d *schema.ResourceData, data *bmcPowerResponse) error {
	nodes := parsePowerResponseForInfo(data)

	if err := d.Set("nodes", nodes); err != nil {
		return fmt.Errorf("failed to set nodes: %w", err)
	}

	return nil
}

// parsePowerResponseForInfo extracts node power status from API response
// Handles both legacy format and new BMC firmware format (2.3.4+)
func parsePowerResponseForInfo(data *bmcPowerResponse) map[string]interface{} {
	nodes := make(map[string]interface{})

	// Initialize all nodes to false
	nodes["node1"] = false
	nodes["node2"] = false
	nodes["node3"] = false
	nodes["node4"] = false

	// Try parsing as new format first: [{"result": [{"node1": "1", ...}]}]
	var newFormat []map[string]interface{}
	if err := json.Unmarshal(data.Response, &newFormat); err == nil {
		for _, item := range newFormat {
			// Check for "result" array in the response
			if result, ok := item["result"].([]interface{}); ok {
				for _, r := range result {
					if nodeMap, ok := r.(map[string]interface{}); ok {
						for nodeName, value := range nodeMap {
							nodes[nodeName] = parsePowerValueForInfo(value)
						}
					}
				}
			}
		}
		return nodes
	}

	// Fall back to legacy format: [[nodeName, status], [nodeName, status], ...]
	var legacyFormat [][]interface{}
	if err := json.Unmarshal(data.Response, &legacyFormat); err == nil {
		for _, item := range legacyFormat {
			if len(item) >= 2 {
				nodeName, nameOk := item[0].(string)
				if nameOk {
					nodes[nodeName] = parsePowerValueForInfo(item[1])
				}
			}
		}
	}

	return nodes
}

// parsePowerValueForInfo converts various types to boolean power state
func parsePowerValueForInfo(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val == 1
	case int:
		return val == 1
	case string:
		return val == "1" || val == "true" || val == "on"
	}
	return false
}

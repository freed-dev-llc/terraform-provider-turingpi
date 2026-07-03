package provider

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const defaultEndpoint = "https://turingpi.local"

// HTTPClient is the shared HTTP client for all API requests
var HTTPClient = &http.Client{Transport: newRetryTransport(http.DefaultTransport)}

// ProviderConfig holds the configuration for the provider
type ProviderConfig struct {
	Token    string
	Endpoint string
}

func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"username": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("TURINGPI_USERNAME", nil),
				Description: "The username for BMC authentication",
			},
			"password": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("TURINGPI_PASSWORD", nil),
				Description: "The password for BMC authentication",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TURINGPI_ENDPOINT", defaultEndpoint),
				Description: "The BMC API endpoint URL (e.g., https://turingpi.local or https://192.168.1.100)",
			},
			"insecure": {
				Type:        schema.TypeBool,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TURINGPI_INSECURE", false),
				Description: "Skip TLS certificate verification (useful for self-signed or expired certificates)",
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"turingpi_power":          resourcePower(),
			"turingpi_flash":          resourceFlash(),
			"turingpi_node":           resourceNode(),
			"turingpi_usb":            resourceUSB(),
			"turingpi_network_reset":  resourceNetworkReset(),
			"turingpi_bmc_firmware":   resourceBMCFirmware(),
			"turingpi_uart":           resourceUART(),
			"turingpi_bmc_reboot":     resourceBMCReboot(),
			"turingpi_usb_boot":       resourceUSBBoot(),
			"turingpi_node_to_msd":    resourceNodeToMSD(),
			"turingpi_clear_usb_boot": resourceClearUSBBoot(),
			"turingpi_bmc_reload":     resourceBMCReload(),
			"turingpi_k3s_cluster":    resourceK3sCluster(),
			"turingpi_talos_cluster":  resourceTalosCluster(),
		},
		DataSourcesMap: map[string]*schema.Resource{
			"turingpi_info":   dataSourceInfo(),
			"turingpi_usb":    dataSourceUSB(),
			"turingpi_power":  dataSourcePower(),
			"turingpi_uart":   dataSourceUART(),
			"turingpi_sdcard": dataSourceSDCard(),
			"turingpi_about":  dataSourceAbout(),
		},
		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	username := d.Get("username").(string)
	password := d.Get("password").(string)
	endpoint := d.Get("endpoint").(string)
	insecure := d.Get("insecure").(bool)

	// Configure HTTP client with TLS settings
	if insecure {
		HTTPClient = &http.Client{
			Transport: newRetryTransport(&http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}),
		}
	}

	token, err := authenticate(endpoint, username, password)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return &ProviderConfig{
		Token:    token,
		Endpoint: endpoint,
	}, nil
}

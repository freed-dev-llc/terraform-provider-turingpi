package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceTalosCluster() *schema.Resource {
	return &schema.Resource{
		Description: "Deploys a Talos Kubernetes cluster on pre-flashed Turing Pi nodes using talosctl.",
		DeprecationMessage: "turingpi_talos_cluster is deprecated and will be removed in v2.0.0. " +
			"Use the terraform-turingpi-modules/talos-cluster module instead. " +
			"See: https://github.com/freed-dev-llc/terraform-turingpi-modules",
		CreateContext: resourceTalosClusterCreate,
		ReadContext:   resourceTalosClusterRead,
		UpdateContext: resourceTalosClusterUpdate,
		DeleteContext: resourceTalosClusterDelete,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the Talos cluster.",
			},
			"cluster_endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Kubernetes API endpoint URL (e.g., https://10.10.88.73:6443).",
			},
			"talos_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Talos version for reference (not used in provisioning).",
			},
			"kubernetes_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Kubernetes version for reference.",
			},
			"install_disk": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "/dev/mmcblk0",
				ForceNew:    true,
				Description: "Install disk for Talos (default: /dev/mmcblk0 for eMMC).",
			},
			"control_plane": {
				Type:        schema.TypeList,
				Required:    true,
				MinItems:    1,
				ForceNew:    true,
				Description: "Control plane node configuration.",
				Elem:        talosNodeSchema(),
			},
			"worker": {
				Type:        schema.TypeList,
				Optional:    true,
				ForceNew:    true,
				Description: "Worker node configurations.",
				Elem:        talosNodeSchema(),
			},
			"allow_scheduling_on_control_plane": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				ForceNew:    true,
				Description: "Allow scheduling workloads on control plane nodes.",
			},
			"metallb": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "MetalLB load balancer configuration.",
				Elem:        metallbSchema(),
			},
			"ingress": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "NGINX Ingress controller configuration.",
				Elem:        ingressSchema(),
			},
			"bootstrap_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     600,
				Description: "Timeout in seconds for cluster bootstrap operations.",
			},
			"kubeconfig_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to write the kubeconfig file.",
			},
			"talosconfig_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to write the talosconfig file.",
			},
			"secrets_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to write the cluster secrets file (for backup).",
			},
			// Computed outputs
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Kubeconfig content for accessing the cluster.",
			},
			"talosconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Talosconfig content for talosctl CLI.",
			},
			"secrets_yaml": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Cluster secrets (PKI) in YAML format.",
			},
			"api_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kubernetes API server endpoint.",
			},
			"cluster_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current status of the cluster (bootstrapping, ready, degraded).",
			},
		},
	}
}

func talosNodeSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "IP address or hostname of the node.",
			},
			"hostname": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Hostname to assign to the node (defaults to turing-cp-N or turing-w-N).",
			},
		},
	}
}

func extractTalosNodeConfig(data map[string]interface{}) TalosNodeConfig {
	config := TalosNodeConfig{}

	if v, ok := data["host"].(string); ok {
		config.Host = v
	}
	if v, ok := data["hostname"].(string); ok {
		config.Hostname = v
	}

	return config
}

func extractTalosClusterConfig(d *schema.ResourceData) TalosClusterConfig {
	cfg := TalosClusterConfig{
		Name:                d.Get("name").(string),
		ClusterEndpoint:     d.Get("cluster_endpoint").(string),
		KubernetesVersion:   d.Get("kubernetes_version").(string),
		InstallDisk:         d.Get("install_disk").(string),
		AllowSchedulingOnCP: d.Get("allow_scheduling_on_control_plane").(bool),
		BootstrapTimeout:    time.Duration(d.Get("bootstrap_timeout").(int)) * time.Second,
	}

	// Extract control plane nodes
	if v, ok := d.GetOk("control_plane"); ok {
		for _, cp := range v.([]interface{}) {
			cfg.ControlPlanes = append(cfg.ControlPlanes, extractTalosNodeConfig(cp.(map[string]interface{})))
		}
	}

	// Extract worker nodes
	if v, ok := d.GetOk("worker"); ok {
		for _, w := range v.([]interface{}) {
			cfg.Workers = append(cfg.Workers, extractTalosNodeConfig(w.(map[string]interface{})))
		}
	}

	return cfg
}

func resourceTalosClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	cfg := extractTalosClusterConfig(d)

	// Create provisioner
	provisioner, err := NewTalosProvisioner()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to create Talos provisioner: %w", err))
	}
	defer func() { _ = provisioner.Cleanup() }()

	// Set initial status
	if err := d.Set("cluster_status", "bootstrapping"); err != nil {
		return diag.FromErr(err)
	}

	// Provision the cluster
	state, err := provisioner.ProvisionCluster(ctx, cfg)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to provision cluster: %w", err))
	}

	// Set computed values
	if err := d.Set("kubeconfig", state.Kubeconfig); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("talosconfig", state.Talosconfig); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("secrets_yaml", state.SecretsYAML); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("api_endpoint", state.APIEndpoint); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set("cluster_status", state.ClusterStatus); err != nil {
		return diag.FromErr(err)
	}

	// Write kubeconfig to file if path specified
	if kubeconfigPath := d.Get("kubeconfig_path").(string); kubeconfigPath != "" && state.Kubeconfig != "" {
		if err := os.WriteFile(kubeconfigPath, []byte(state.Kubeconfig), 0600); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Failed to write kubeconfig file",
				Detail:   fmt.Sprintf("Could not write kubeconfig to %s: %v", kubeconfigPath, err),
			})
		}
	}

	// Write talosconfig to file if path specified
	if talosconfigPath := d.Get("talosconfig_path").(string); talosconfigPath != "" && state.Talosconfig != "" {
		if err := os.WriteFile(talosconfigPath, []byte(state.Talosconfig), 0600); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Failed to write talosconfig file",
				Detail:   fmt.Sprintf("Could not write talosconfig to %s: %v", talosconfigPath, err),
			})
		}
	}

	// Write secrets to file if path specified
	if secretsPath := d.Get("secrets_path").(string); secretsPath != "" && state.SecretsYAML != "" {
		if err := os.WriteFile(secretsPath, []byte(state.SecretsYAML), 0600); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Failed to write secrets file",
				Detail:   fmt.Sprintf("Could not write secrets to %s: %v", secretsPath, err),
			})
		}
	}

	// Deploy addons if enabled
	if state.Kubeconfig != "" {
		// Create temp kubeconfig file for addon deployment
		kubeconfigFile, err := os.CreateTemp("", "kubeconfig-*")
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to create temp kubeconfig: %w", err))
		}
		defer func() { _ = os.Remove(kubeconfigFile.Name()) }()

		if _, err := kubeconfigFile.WriteString(state.Kubeconfig); err != nil {
			return diag.FromErr(fmt.Errorf("failed to write temp kubeconfig: %w", err))
		}
		if err := kubeconfigFile.Close(); err != nil {
			return diag.FromErr(fmt.Errorf("failed to close temp kubeconfig: %w", err))
		}

		// Deploy MetalLB if enabled
		if metallbList := d.Get("metallb").([]interface{}); len(metallbList) > 0 {
			metallbConfig := metallbList[0].(map[string]interface{})
			if enabled, ok := metallbConfig["enabled"].(bool); ok && enabled {
				ipRange := metallbConfig["ip_range"].(string)
				if err := deployMetalLB(ctx, kubeconfigFile.Name(), ipRange); err != nil {
					diags = append(diags, diag.Diagnostic{
						Severity: diag.Warning,
						Summary:  "Failed to deploy MetalLB",
						Detail:   fmt.Sprintf("MetalLB deployment failed: %v", err),
					})
				}
			}
		}

		// Deploy Ingress if enabled
		if ingressList := d.Get("ingress").([]interface{}); len(ingressList) > 0 {
			ingressConfig := ingressList[0].(map[string]interface{})
			if enabled, ok := ingressConfig["enabled"].(bool); ok && enabled {
				ingressIP := ""
				if ip, ok := ingressConfig["ip"].(string); ok {
					ingressIP = ip
				} else if metallbList := d.Get("metallb").([]interface{}); len(metallbList) > 0 {
					// Use first IP from MetalLB range
					metallbConfig := metallbList[0].(map[string]interface{})
					if ipRange, ok := metallbConfig["ip_range"].(string); ok {
						parts := splitIPRange(ipRange)
						if len(parts) > 0 {
							ingressIP = parts[0]
						}
					}
				}

				if ingressIP != "" {
					if err := deployNginxIngress(ctx, kubeconfigFile.Name(), ingressIP); err != nil {
						diags = append(diags, diag.Diagnostic{
							Severity: diag.Warning,
							Summary:  "Failed to deploy NGINX Ingress",
							Detail:   fmt.Sprintf("Ingress deployment failed: %v", err),
						})
					}
				}
			}
		}
	}

	d.SetId(cfg.Name)

	return diags
}

func resourceTalosClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// Get stored talosconfig
	talosconfig := d.Get("talosconfig").(string)
	if talosconfig == "" {
		// No talosconfig means cluster doesn't exist
		d.SetId("")
		return diags
	}

	// Get control plane IP
	controlPlanes := d.Get("control_plane").([]interface{})
	if len(controlPlanes) == 0 {
		d.SetId("")
		return diags
	}

	cpConfig := controlPlanes[0].(map[string]interface{})
	cpHost := cpConfig["host"].(string)

	// Create provisioner to check health
	provisioner, err := NewTalosProvisioner()
	if err != nil {
		// If talosctl not available, just return current state
		return diags
	}
	defer func() { _ = provisioner.Cleanup() }()

	// Check cluster health
	status, err := provisioner.CheckClusterHealth(talosconfig, cpHost)
	if err != nil {
		status = "unknown"
	}

	if err := d.Set("cluster_status", status); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func resourceTalosClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Most changes require ForceNew, so this is mostly a no-op
	// Only addon changes can be applied without recreation

	var diags diag.Diagnostics

	// Check if addon configuration changed
	if d.HasChange("metallb") || d.HasChange("ingress") {
		kubeconfig := d.Get("kubeconfig").(string)
		if kubeconfig == "" {
			return diag.Errorf("no kubeconfig available for addon updates")
		}

		// Create temp kubeconfig file
		kubeconfigFile, err := os.CreateTemp("", "kubeconfig-*")
		if err != nil {
			return diag.FromErr(err)
		}
		defer func() { _ = os.Remove(kubeconfigFile.Name()) }()

		if _, err := kubeconfigFile.WriteString(kubeconfig); err != nil {
			return diag.FromErr(err)
		}
		if err := kubeconfigFile.Close(); err != nil {
			return diag.FromErr(err)
		}

		// Deploy/update MetalLB if changed
		if d.HasChange("metallb") {
			if metallbList := d.Get("metallb").([]interface{}); len(metallbList) > 0 {
				metallbConfig := metallbList[0].(map[string]interface{})
				if enabled, ok := metallbConfig["enabled"].(bool); ok && enabled {
					ipRange := metallbConfig["ip_range"].(string)
					if err := deployMetalLB(ctx, kubeconfigFile.Name(), ipRange); err != nil {
						diags = append(diags, diag.Diagnostic{
							Severity: diag.Warning,
							Summary:  "Failed to update MetalLB",
							Detail:   err.Error(),
						})
					}
				}
			}
		}

		// Deploy/update Ingress if changed
		if d.HasChange("ingress") {
			if ingressList := d.Get("ingress").([]interface{}); len(ingressList) > 0 {
				ingressConfig := ingressList[0].(map[string]interface{})
				if enabled, ok := ingressConfig["enabled"].(bool); ok && enabled {
					ingressIP := ""
					if ip, ok := ingressConfig["ip"].(string); ok {
						ingressIP = ip
					}
					if ingressIP != "" {
						if err := deployNginxIngress(ctx, kubeconfigFile.Name(), ingressIP); err != nil {
							diags = append(diags, diag.Diagnostic{
								Severity: diag.Warning,
								Summary:  "Failed to update NGINX Ingress",
								Detail:   err.Error(),
							})
						}
					}
				}
			}
		}
	}

	return diags
}

func resourceTalosClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	// Get stored talosconfig
	talosconfig := d.Get("talosconfig").(string)
	if talosconfig == "" {
		// No talosconfig, nothing to delete
		d.SetId("")
		return diags
	}

	// Extract node IPs
	var controlPlaneIPs, workerIPs []string

	if cpList := d.Get("control_plane").([]interface{}); len(cpList) > 0 {
		for _, cp := range cpList {
			cpConfig := cp.(map[string]interface{})
			controlPlaneIPs = append(controlPlaneIPs, cpConfig["host"].(string))
		}
	}

	if workerList := d.Get("worker").([]interface{}); len(workerList) > 0 {
		for _, w := range workerList {
			wConfig := w.(map[string]interface{})
			workerIPs = append(workerIPs, wConfig["host"].(string))
		}
	}

	// Create provisioner
	provisioner, err := NewTalosProvisioner()
	if err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Could not create Talos provisioner",
			Detail:   fmt.Sprintf("talosctl may not be available: %v", err),
		})
		d.SetId("")
		return diags
	}
	defer func() { _ = provisioner.Cleanup() }()

	// Destroy the cluster
	if err := provisioner.DestroyCluster(talosconfig, controlPlaneIPs, workerIPs); err != nil {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Cluster destruction may be incomplete",
			Detail:   fmt.Sprintf("Some nodes may not have been reset: %v", err),
		})
	}

	// Clean up local files
	if kubeconfigPath := d.Get("kubeconfig_path").(string); kubeconfigPath != "" {
		_ = os.Remove(kubeconfigPath)
	}
	if talosconfigPath := d.Get("talosconfig_path").(string); talosconfigPath != "" {
		_ = os.Remove(talosconfigPath)
	}
	if secretsPath := d.Get("secrets_path").(string); secretsPath != "" {
		_ = os.Remove(secretsPath)
	}

	d.SetId("")
	return diags
}

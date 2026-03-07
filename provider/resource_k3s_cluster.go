package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceK3sCluster() *schema.Resource {
	return &schema.Resource{
		Description: "Deploys a K3s Kubernetes cluster on pre-flashed Turing Pi nodes",
		DeprecationMessage: "turingpi_k3s_cluster is deprecated and will be removed in v2.0.0. " +
			"Use the terraform-turingpi-modules/k3s-cluster module instead. " +
			"See: https://github.com/freed-dev-llc/terraform-turingpi-modules",
		CreateContext: resourceK3sClusterCreate,
		ReadContext:   resourceK3sClusterRead,
		UpdateContext: resourceK3sClusterUpdate,
		DeleteContext: resourceK3sClusterDelete,
		Importer: &schema.ResourceImporter{
			StateContext: resourceK3sClusterImport,
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the K3s cluster",
			},
			"k3s_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "K3s version to install (e.g., v1.31.4+k3s1). Empty for latest stable.",
			},
			"cluster_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				Description: "Cluster token for node authentication. Auto-generated if not provided.",
			},
			"control_plane": {
				Type:        schema.TypeList,
				Required:    true,
				MaxItems:    1,
				Description: "Control plane node configuration",
				Elem:        k3sNodeSchema(),
			},
			"worker": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Worker node configurations",
				Elem:        k3sNodeSchema(),
			},
			"pod_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "10.244.0.0/16",
				Description: "CIDR for pod network",
			},
			"service_cidr": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "10.96.0.0/12",
				Description: "CIDR for service network",
			},
			"metallb": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "MetalLB load balancer configuration",
				Elem:        metallbSchema(),
			},
			"ingress": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "NGINX Ingress controller configuration",
				Elem:        ingressSchema(),
			},
			"install_timeout": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     600,
				Description: "Timeout in seconds for K3s installation (default 10 minutes)",
			},
			"kubeconfig_path": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Path to write the kubeconfig file",
			},
			// Computed outputs
			"kubeconfig": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Kubeconfig content for accessing the cluster",
			},
			"api_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Kubernetes API endpoint URL",
			},
			"node_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Node token for joining additional nodes",
			},
			"cluster_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current cluster status (bootstrapping, ready, degraded)",
			},
		},
	}
}

func k3sNodeSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "IP address or hostname of the node",
			},
			"ssh_user": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "SSH username for connecting to the node",
			},
			"ssh_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "SSH private key content for authentication",
			},
			"ssh_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "SSH password for authentication (ssh_key is preferred)",
			},
			"ssh_port": {
				Type:        schema.TypeInt,
				Optional:    true,
				Default:     22,
				Description: "SSH port number",
			},
		},
	}
}

func metallbSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable MetalLB deployment",
			},
			"ip_range": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "IP address range for MetalLB (e.g., 10.10.88.80-10.10.88.89)",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "MetalLB chart version (empty for latest)",
			},
		},
	}
}

func ingressSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"enabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Enable NGINX Ingress controller deployment",
			},
			"ip": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "LoadBalancer IP for ingress (uses first MetalLB IP if not set)",
			},
			"version": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "NGINX Ingress chart version (empty for latest)",
			},
		},
	}
}

// extractNodeConfig extracts NodeConfig from schema data
func extractNodeConfig(data map[string]interface{}) NodeConfig {
	config := NodeConfig{
		Host:    data["host"].(string),
		SSHUser: data["ssh_user"].(string),
		SSHPort: data["ssh_port"].(int),
	}
	if v, ok := data["ssh_key"].(string); ok && v != "" {
		config.SSHKey = []byte(v)
	}
	if v, ok := data["ssh_password"].(string); ok {
		config.SSHPassword = v
	}
	return config
}

// extractClusterConfig extracts ClusterConfig from ResourceData
func extractClusterConfig(d *schema.ResourceData) ClusterConfig {
	cfg := ClusterConfig{
		Name:         d.Get("name").(string),
		K3sVersion:   d.Get("k3s_version").(string),
		ClusterToken: d.Get("cluster_token").(string),
		PodCIDR:      d.Get("pod_cidr").(string),
		ServiceCIDR:  d.Get("service_cidr").(string),
	}

	// Extract control plane
	if v, ok := d.GetOk("control_plane"); ok {
		cpList := v.([]interface{})
		if len(cpList) > 0 {
			cfg.ControlPlane = extractNodeConfig(cpList[0].(map[string]interface{}))
		}
	}

	// Extract workers
	if v, ok := d.GetOk("worker"); ok {
		workerList := v.([]interface{})
		for _, w := range workerList {
			cfg.Workers = append(cfg.Workers, extractNodeConfig(w.(map[string]interface{})))
		}
	}

	return cfg
}

func resourceK3sClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	cfg := extractClusterConfig(d)
	provisioner := NewK3sProvisioner()
	timeout := time.Duration(d.Get("install_timeout").(int)) * time.Second

	tflog.Info(ctx, "Starting K3s cluster creation", map[string]interface{}{
		"cluster_name":  cfg.Name,
		"control_plane": cfg.ControlPlane.Host,
		"worker_count":  len(cfg.Workers),
		"timeout":       timeout.String(),
	})

	// Set status to bootstrapping
	if err := d.Set("cluster_status", "bootstrapping"); err != nil {
		return diag.FromErr(err)
	}

	// 1. Generate cluster token if not provided
	if cfg.ClusterToken == "" {
		cfg.ClusterToken = GenerateClusterToken()
		if err := d.Set("cluster_token", cfg.ClusterToken); err != nil {
			return diag.FromErr(err)
		}
		tflog.Debug(ctx, "Generated cluster token")
	}

	// 2. Install K3s server on control plane
	tflog.Info(ctx, "Installing K3s server on control plane", map[string]interface{}{
		"host":    cfg.ControlPlane.Host,
		"version": cfg.K3sVersion,
	})
	if err := provisioner.InstallK3sServer(ctx, cfg.ControlPlane, cfg, timeout); err != nil {
		return diag.FromErr(fmt.Errorf("failed to install K3s server: %w", err))
	}
	tflog.Info(ctx, "K3s server installation complete")

	// 3. Get node token and kubeconfig
	nodeToken, err := provisioner.GetNodeToken(cfg.ControlPlane)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get node token: %w", err))
	}
	if err := d.Set("node_token", nodeToken); err != nil {
		return diag.FromErr(err)
	}

	kubeconfig, err := provisioner.GetKubeconfig(cfg.ControlPlane)
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to get kubeconfig: %w", err))
	}
	if err := d.Set("kubeconfig", kubeconfig); err != nil {
		return diag.FromErr(err)
	}

	apiEndpoint := fmt.Sprintf("https://%s:6443", cfg.ControlPlane.Host)
	if err := d.Set("api_endpoint", apiEndpoint); err != nil {
		return diag.FromErr(err)
	}

	// 4. Write kubeconfig to file if path specified
	if kubeconfigPath := d.Get("kubeconfig_path").(string); kubeconfigPath != "" {
		if err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0600); err != nil {
			return diag.FromErr(fmt.Errorf("failed to write kubeconfig to %s: %w", kubeconfigPath, err))
		}
	}

	// 5. Install K3s agents on workers
	serverURL := apiEndpoint
	for i, worker := range cfg.Workers {
		tflog.Info(ctx, "Installing K3s agent on worker", map[string]interface{}{
			"host":         worker.Host,
			"worker_index": i + 1,
			"total":        len(cfg.Workers),
		})
		if err := provisioner.InstallK3sAgent(ctx, worker, serverURL, nodeToken, cfg.K3sVersion, timeout); err != nil {
			return diag.FromErr(fmt.Errorf("failed to install K3s agent on %s: %w", worker.Host, err))
		}

		// Wait for node to be ready
		tflog.Debug(ctx, "Waiting for worker node to be ready", map[string]interface{}{
			"host": worker.Host,
		})
		if err := provisioner.WaitForNodeReady(cfg.ControlPlane, worker.Host, timeout); err != nil {
			return diag.FromErr(fmt.Errorf("worker %s failed to become ready: %w", worker.Host, err))
		}
		tflog.Info(ctx, "Worker node ready", map[string]interface{}{
			"host": worker.Host,
		})
	}

	// 6. Deploy MetalLB if enabled
	if v, ok := d.GetOk("metallb"); ok {
		metallbList := v.([]interface{})
		if len(metallbList) > 0 {
			metallbConfig := metallbList[0].(map[string]interface{})
			if metallbConfig["enabled"].(bool) {
				ipRange := metallbConfig["ip_range"].(string)
				kubeconfigPath := d.Get("kubeconfig_path").(string)

				tflog.Info(ctx, "Deploying MetalLB", map[string]interface{}{
					"ip_range": ipRange,
				})

				// Use a temp file if no path specified
				if kubeconfigPath == "" {
					tmpFile, err := os.CreateTemp("", "kubeconfig-*")
					if err != nil {
						return diag.FromErr(fmt.Errorf("failed to create temp kubeconfig: %w", err))
					}
					kubeconfigPath = tmpFile.Name()
					defer func() { _ = os.Remove(kubeconfigPath) }()
					if err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0600); err != nil {
						return diag.FromErr(err)
					}
				}

				if err := deployMetalLB(ctx, kubeconfigPath, ipRange); err != nil {
					return diag.FromErr(fmt.Errorf("failed to deploy MetalLB: %w", err))
				}
				tflog.Info(ctx, "MetalLB deployment complete", map[string]interface{}{
					"ip_range": ipRange,
				})
			}
		}
	}

	// 7. Deploy NGINX Ingress if enabled
	if v, ok := d.GetOk("ingress"); ok {
		ingressList := v.([]interface{})
		if len(ingressList) > 0 {
			ingressConfig := ingressList[0].(map[string]interface{})
			if ingressConfig["enabled"].(bool) {
				ingressIP := ingressConfig["ip"].(string)

				// If no ingress IP specified, try to use first MetalLB IP
				if ingressIP == "" {
					if metallbList, ok := d.GetOk("metallb"); ok {
						metallbConfigs := metallbList.([]interface{})
						if len(metallbConfigs) > 0 {
							ipRange := metallbConfigs[0].(map[string]interface{})["ip_range"].(string)
							// Extract first IP from range (e.g., "10.10.88.80-10.10.88.89" -> "10.10.88.80")
							if idx := len(ipRange); idx > 0 {
								parts := splitIPRange(ipRange)
								if len(parts) > 0 {
									ingressIP = parts[0]
								}
							}
						}
					}
				}

				tflog.Info(ctx, "Deploying NGINX Ingress controller", map[string]interface{}{
					"load_balancer_ip": ingressIP,
				})

				kubeconfigPath := d.Get("kubeconfig_path").(string)
				if kubeconfigPath == "" {
					tmpFile, err := os.CreateTemp("", "kubeconfig-*")
					if err != nil {
						return diag.FromErr(fmt.Errorf("failed to create temp kubeconfig: %w", err))
					}
					kubeconfigPath = tmpFile.Name()
					defer func() { _ = os.Remove(kubeconfigPath) }()
					if err := os.WriteFile(kubeconfigPath, []byte(kubeconfig), 0600); err != nil {
						return diag.FromErr(err)
					}
				}

				if err := deployNginxIngress(ctx, kubeconfigPath, ingressIP); err != nil {
					return diag.FromErr(fmt.Errorf("failed to deploy NGINX Ingress: %w", err))
				}
				tflog.Info(ctx, "NGINX Ingress deployment complete")
			}
		}
	}

	d.SetId(cfg.Name)
	if err := d.Set("cluster_status", "ready"); err != nil {
		return diag.FromErr(err)
	}

	tflog.Info(ctx, "K3s cluster creation complete", map[string]interface{}{
		"cluster_name": cfg.Name,
		"api_endpoint": apiEndpoint,
	})

	return diags
}

func resourceK3sClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	cfg := extractClusterConfig(d)
	provisioner := NewK3sProvisioner()

	// Check if K3s is still installed on control plane
	installed, err := provisioner.CheckK3sInstalled(cfg.ControlPlane)
	if err != nil || !installed {
		// K3s not installed, resource has been deleted externally
		d.SetId("")
		return diags
	}

	// Get cluster status by checking node count
	nodes, err := provisioner.GetClusterNodes(cfg.ControlPlane)
	if err != nil {
		if err := d.Set("cluster_status", "degraded"); err != nil {
			return diag.FromErr(err)
		}
		return diags
	}

	expectedNodes := 1 + len(cfg.Workers)
	if len(nodes) >= expectedNodes {
		if err := d.Set("cluster_status", "ready"); err != nil {
			return diag.FromErr(err)
		}
	} else {
		if err := d.Set("cluster_status", "degraded"); err != nil {
			return diag.FromErr(err)
		}
	}

	// Refresh kubeconfig
	kubeconfig, err := provisioner.GetKubeconfig(cfg.ControlPlane)
	if err == nil {
		if err := d.Set("kubeconfig", kubeconfig); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func resourceK3sClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// For now, updates are handled by detecting changes and re-applying
	// Full update logic can be added later (e.g., adding/removing workers)

	if d.HasChange("worker") {
		// Handle worker changes
		old, new := d.GetChange("worker")
		oldWorkers := old.([]interface{})
		newWorkers := new.([]interface{})

		cfg := extractClusterConfig(d)
		provisioner := NewK3sProvisioner()
		timeout := time.Duration(d.Get("install_timeout").(int)) * time.Second

		nodeToken, err := provisioner.GetNodeToken(cfg.ControlPlane)
		if err != nil {
			return diag.FromErr(err)
		}

		serverURL := fmt.Sprintf("https://%s:6443", cfg.ControlPlane.Host)

		// Install new workers
		if len(newWorkers) > len(oldWorkers) {
			for i := len(oldWorkers); i < len(newWorkers); i++ {
				worker := extractNodeConfig(newWorkers[i].(map[string]interface{}))
				if err := provisioner.InstallK3sAgent(ctx, worker, serverURL, nodeToken, cfg.K3sVersion, timeout); err != nil {
					return diag.FromErr(err)
				}
				if err := provisioner.WaitForNodeReady(cfg.ControlPlane, worker.Host, timeout); err != nil {
					return diag.FromErr(err)
				}
			}
		}

		// Note: Removing workers would require additional logic to drain and remove nodes
	}

	return resourceK3sClusterRead(ctx, d, meta)
}

func resourceK3sClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	cfg := extractClusterConfig(d)
	provisioner := NewK3sProvisioner()

	// Uninstall agents first
	for _, worker := range cfg.Workers {
		if err := provisioner.UninstallK3sAgent(worker); err != nil {
			// Log error but continue with other nodes
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("Failed to uninstall K3s agent on %s", worker.Host),
				Detail:   err.Error(),
			})
		}
	}

	// Uninstall server
	if err := provisioner.UninstallK3sServer(cfg.ControlPlane); err != nil {
		return diag.FromErr(fmt.Errorf("failed to uninstall K3s server: %w", err))
	}

	// Remove kubeconfig file if it was created
	if kubeconfigPath := d.Get("kubeconfig_path").(string); kubeconfigPath != "" {
		_ = os.Remove(kubeconfigPath)
	}

	d.SetId("")
	return diags
}

// resourceK3sClusterImport imports an existing K3s cluster into Terraform state
// Import format: "cluster_name:control_plane_host:ssh_user:ssh_key_path"
// Example: terraform import turingpi_k3s_cluster.mycluster "mycluster:10.10.88.73:root:/home/user/.ssh/id_ed25519"
func resourceK3sClusterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	tflog.Info(ctx, "Importing K3s cluster")

	// Parse import ID: cluster_name:control_plane_host:ssh_user:ssh_key_path
	idParts := strings.Split(d.Id(), ":")
	if len(idParts) < 4 {
		return nil, fmt.Errorf("invalid import ID format. Expected: cluster_name:control_plane_host:ssh_user:ssh_key_path")
	}

	clusterName := idParts[0]
	controlPlaneHost := idParts[1]
	sshUser := idParts[2]
	sshKeyPath := idParts[3]

	// Read SSH key
	sshKey, err := os.ReadFile(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SSH key from %s: %w", sshKeyPath, err)
	}

	// Create node config for control plane
	controlPlane := NodeConfig{
		Host:    controlPlaneHost,
		SSHUser: sshUser,
		SSHKey:  sshKey,
		SSHPort: 22,
	}

	provisioner := NewK3sProvisioner()

	// Verify K3s is installed on control plane
	installed, err := provisioner.CheckK3sInstalled(controlPlane)
	if err != nil {
		return nil, fmt.Errorf("failed to check K3s installation: %w", err)
	}
	if !installed {
		return nil, fmt.Errorf("K3s is not installed on %s", controlPlaneHost)
	}

	tflog.Info(ctx, "K3s installation found on control plane", map[string]interface{}{
		"host": controlPlaneHost,
	})

	// Get kubeconfig
	kubeconfig, err := provisioner.GetKubeconfig(controlPlane)
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	// Get node token
	nodeToken, err := provisioner.GetNodeToken(controlPlane)
	if err != nil {
		return nil, fmt.Errorf("failed to get node token: %w", err)
	}

	// Get K3s version
	version, err := provisioner.GetK3sVersion(controlPlane)
	if err != nil {
		tflog.Warn(ctx, "Failed to get K3s version", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Get cluster nodes to determine workers
	nodes, err := provisioner.GetClusterNodes(controlPlane)
	if err != nil {
		tflog.Warn(ctx, "Failed to get cluster nodes", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Set resource ID
	d.SetId(clusterName)

	// Set basic attributes
	if err := d.Set("name", clusterName); err != nil {
		return nil, err
	}
	if err := d.Set("kubeconfig", kubeconfig); err != nil {
		return nil, err
	}
	if err := d.Set("node_token", nodeToken); err != nil {
		return nil, err
	}
	if err := d.Set("api_endpoint", fmt.Sprintf("https://%s:6443", controlPlaneHost)); err != nil {
		return nil, err
	}
	if version != "" {
		// Extract just the version number (e.g., "v1.31.4+k3s1" from "k3s version v1.31.4+k3s1 (xxx)")
		versionParts := strings.Fields(version)
		for _, part := range versionParts {
			if strings.HasPrefix(part, "v") {
				if err := d.Set("k3s_version", part); err != nil {
					return nil, err
				}
				break
			}
		}
	}

	// Set control plane configuration
	controlPlaneConfig := []interface{}{
		map[string]interface{}{
			"host":     controlPlaneHost,
			"ssh_user": sshUser,
			"ssh_key":  string(sshKey),
			"ssh_port": 22,
		},
	}
	if err := d.Set("control_plane", controlPlaneConfig); err != nil {
		return nil, err
	}

	// Determine cluster status
	status := "ready"
	if len(nodes) == 0 {
		status = "degraded"
	}
	if err := d.Set("cluster_status", status); err != nil {
		return nil, err
	}

	tflog.Info(ctx, "K3s cluster imported successfully", map[string]interface{}{
		"cluster_name": clusterName,
		"node_count":   len(nodes),
		"status":       status,
	})

	return []*schema.ResourceData{d}, nil
}

// splitIPRange extracts the start IP from an IP range string like "10.10.88.80-10.10.88.89"
func splitIPRange(ipRange string) []string {
	parts := make([]string, 0)
	for i := 0; i < len(ipRange); i++ {
		if ipRange[i] == '-' {
			parts = append(parts, ipRange[:i])
			if i+1 < len(ipRange) {
				parts = append(parts, ipRange[i+1:])
			}
			break
		}
	}
	if len(parts) == 0 && ipRange != "" {
		parts = append(parts, ipRange)
	}
	return parts
}

// deployMetalLB deploys MetalLB using Helm and creates IPAddressPool and L2Advertisement
func deployMetalLB(ctx context.Context, kubeconfigPath, ipRange string) error {
	tflog.Debug(ctx, "Creating Helm client for MetalLB deployment")

	client, err := NewHelmClient(kubeconfigPath, "metallb-system")
	if err != nil {
		return fmt.Errorf("failed to create Helm client: %w", err)
	}

	// Add MetalLB repo
	tflog.Debug(ctx, "Adding MetalLB Helm repository")
	if err := client.AddRepository("metallb", "https://metallb.github.io/metallb"); err != nil {
		return fmt.Errorf("failed to add MetalLB repo: %w", err)
	}

	// Install MetalLB chart
	tflog.Debug(ctx, "Installing MetalLB Helm chart")
	spec := &ChartSpec{
		ReleaseName:     "metallb",
		ChartName:       "metallb/metallb",
		Namespace:       "metallb-system",
		CreateNamespace: true,
		Wait:            true,
		Timeout:         5 * time.Minute,
	}

	if _, err := client.InstallOrUpgradeChart(ctx, spec); err != nil {
		return fmt.Errorf("failed to install MetalLB chart: %w", err)
	}

	// Wait for MetalLB CRDs to be available
	tflog.Debug(ctx, "Waiting for MetalLB CRDs to be available")
	if err := waitForMetalLBReady(ctx, kubeconfigPath); err != nil {
		return fmt.Errorf("MetalLB CRDs not ready: %w", err)
	}

	// Create IPAddressPool and L2Advertisement
	tflog.Debug(ctx, "Creating IPAddressPool and L2Advertisement", map[string]interface{}{
		"ip_range": ipRange,
	})
	if err := applyMetalLBConfig(ctx, kubeconfigPath, ipRange); err != nil {
		return fmt.Errorf("failed to create MetalLB configuration: %w", err)
	}

	tflog.Debug(ctx, "MetalLB deployment and configuration complete")
	return nil
}

// waitForMetalLBReady waits for MetalLB CRDs and pods to be ready
func waitForMetalLBReady(ctx context.Context, kubeconfigPath string) error {
	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	k8sClient, err := NewK8sClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Wait for IPAddressPool CRD to exist (indicates MetalLB is ready)
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Check if CRD exists by trying to list IPAddressPools
		_, err := k8sClient.RunKubectl("get", "crd", "ipaddresspools.metallb.io")
		if err == nil {
			// CRD exists, also check if controller pod is ready
			output, err := k8sClient.RunKubectl("get", "pods", "-n", "metallb-system", "-l", "app.kubernetes.io/component=controller", "-o", "jsonpath={.items[0].status.phase}")
			if err == nil && strings.TrimSpace(output) == "Running" {
				return nil
			}
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("timeout waiting for MetalLB to be ready")
}

// applyMetalLBConfig creates the IPAddressPool and L2Advertisement resources
func applyMetalLBConfig(ctx context.Context, kubeconfigPath, ipRange string) error {
	kubeconfig, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	k8sClient, err := NewK8sClient(kubeconfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Create IPAddressPool manifest
	ipAddressPoolManifest := fmt.Sprintf(`apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: default-pool
  namespace: metallb-system
spec:
  addresses:
  - %s
`, ipRange)

	// Create L2Advertisement manifest
	l2AdvertisementManifest := `apiVersion: metallb.io/v1beta1
kind: L2Advertisement
metadata:
  name: default-l2
  namespace: metallb-system
spec:
  ipAddressPools:
  - default-pool
`

	// Apply IPAddressPool
	if err := k8sClient.ApplyManifest(ipAddressPoolManifest); err != nil {
		return fmt.Errorf("failed to create IPAddressPool: %w", err)
	}

	// Apply L2Advertisement
	if err := k8sClient.ApplyManifest(l2AdvertisementManifest); err != nil {
		return fmt.Errorf("failed to create L2Advertisement: %w", err)
	}

	return nil
}

// deployNginxIngress deploys NGINX Ingress controller using Helm
func deployNginxIngress(ctx context.Context, kubeconfigPath, loadBalancerIP string) error {
	client, err := NewHelmClient(kubeconfigPath, "ingress-nginx")
	if err != nil {
		return fmt.Errorf("failed to create Helm client: %w", err)
	}

	// Add ingress-nginx repo
	if err := client.AddRepository("ingress-nginx", "https://kubernetes.github.io/ingress-nginx"); err != nil {
		return fmt.Errorf("failed to add ingress-nginx repo: %w", err)
	}

	// Build values YAML
	valuesYaml := `controller:
  ingressClassResource:
    default: true
  service:
    type: LoadBalancer`

	if loadBalancerIP != "" {
		valuesYaml = fmt.Sprintf(`controller:
  ingressClassResource:
    default: true
  service:
    type: LoadBalancer
    loadBalancerIP: "%s"`, loadBalancerIP)
	}

	// Install ingress-nginx chart
	spec := &ChartSpec{
		ReleaseName:     "ingress-nginx",
		ChartName:       "ingress-nginx/ingress-nginx",
		Namespace:       "ingress-nginx",
		CreateNamespace: true,
		Wait:            true,
		Timeout:         5 * time.Minute,
		ValuesYaml:      valuesYaml,
	}

	if _, err := client.InstallOrUpgradeChart(ctx, spec); err != nil {
		return fmt.Errorf("failed to install ingress-nginx chart: %w", err)
	}

	return nil
}

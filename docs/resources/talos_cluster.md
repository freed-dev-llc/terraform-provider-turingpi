---
page_title: "turingpi_talos_cluster Resource - Turing Pi"
subcategory: ""
description: |-
  Deploys a Talos Kubernetes cluster on pre-flashed Turing Pi nodes using talosctl.
---

# turingpi_talos_cluster (Resource)

~> **Deprecation Warning:** This resource is deprecated and will be removed in v2.0.0. Please migrate to the [terraform-turingpi-modules](https://github.com/freed-dev-llc/terraform-turingpi-modules) repository which uses the native [Talos Terraform Provider](https://registry.terraform.io/providers/siderolabs/talos/latest) and provides separate, composable modules for cluster deployment and addon management.

Deploys a Talos Kubernetes cluster on pre-flashed Turing Pi nodes. This resource uses the `talosctl` CLI for all Talos operations including config generation, bootstrap, and cluster management.

**Prerequisites:**
- `talosctl` must be installed and available in PATH
- Nodes must be pre-flashed with Talos Linux image (use `turingpi_node` resource)
- Nodes must be powered on and accessible on the network

**Note:** Talos Linux uses a mainline kernel which does not include Rockchip NPU drivers. For NPU workloads on RK3588, use `turingpi_k3s_cluster` with Armbian instead.

## Example Usage

### Minimal Single-Node Cluster

```hcl
resource "turingpi_talos_cluster" "cluster" {
  name             = "my-cluster"
  cluster_endpoint = "https://10.10.88.73:6443"

  control_plane {
    host = "10.10.88.73"
  }
}

output "kubeconfig" {
  value     = turingpi_talos_cluster.cluster.kubeconfig
  sensitive = true
}
```

### Multi-Node Cluster with Workers

```hcl
resource "turingpi_talos_cluster" "cluster" {
  name             = "turing-talos"
  cluster_endpoint = "https://10.10.88.73:6443"

  control_plane {
    host     = "10.10.88.73"
    hostname = "turing-cp1"
  }

  worker {
    host     = "10.10.88.74"
    hostname = "turing-w1"
  }

  worker {
    host     = "10.10.88.75"
    hostname = "turing-w2"
  }

  worker {
    host     = "10.10.88.76"
    hostname = "turing-w3"
  }

  kubeconfig_path  = "./kubeconfig"
  talosconfig_path = "./talosconfig"
}
```

### Full Cluster with MetalLB and Ingress

```hcl
resource "turingpi_talos_cluster" "production" {
  name             = "production"
  cluster_endpoint = "https://10.10.88.73:6443"

  control_plane {
    host     = "10.10.88.73"
    hostname = "turing-cp1"
  }

  worker {
    host     = "10.10.88.74"
    hostname = "turing-w1"
  }

  worker {
    host     = "10.10.88.75"
    hostname = "turing-w2"
  }

  worker {
    host     = "10.10.88.76"
    hostname = "turing-w3"
  }

  # Allow pods on control plane (useful for small clusters)
  allow_scheduling_on_control_plane = true

  # MetalLB for LoadBalancer services
  metallb {
    enabled  = true
    ip_range = "10.10.88.80-10.10.88.89"
  }

  # NGINX Ingress controller
  ingress {
    enabled = true
    ip      = "10.10.88.80"
  }

  bootstrap_timeout = 900
  kubeconfig_path   = "./production-kubeconfig"
  talosconfig_path  = "./production-talosconfig"
  secrets_path      = "./production-secrets.yaml"
}

output "api_endpoint" {
  value = turingpi_talos_cluster.production.api_endpoint
}

output "cluster_status" {
  value = turingpi_talos_cluster.production.cluster_status
}
```

### Complete Workflow with Node Flashing

```hcl
# First, flash Talos image to nodes
resource "turingpi_node" "nodes" {
  for_each = toset(["1", "2", "3", "4"])

  node          = tonumber(each.key)
  power_state   = "on"
  firmware_file = "/path/to/talos-metal-arm64.raw"
  boot_check    = true
  boot_check_pattern = "maintenance mode"
  login_prompt_timeout = 300
}

# Then deploy Talos cluster
resource "turingpi_talos_cluster" "cluster" {
  depends_on = [turingpi_node.nodes]

  name             = "turing-cluster"
  cluster_endpoint = "https://10.10.88.73:6443"

  control_plane {
    host     = "10.10.88.73"
    hostname = "turing-cp1"
  }

  worker {
    host     = "10.10.88.74"
    hostname = "turing-w1"
  }

  worker {
    host     = "10.10.88.75"
    hostname = "turing-w2"
  }

  worker {
    host     = "10.10.88.76"
    hostname = "turing-w3"
  }

  metallb {
    enabled  = true
    ip_range = "10.10.88.80-10.10.88.89"
  }

  ingress {
    enabled = true
  }

  kubeconfig_path = "./kubeconfig"
}
```

## Argument Reference

### Required Arguments

- `name` - (Required, String, ForceNew) Name of the Talos cluster. Used for identification and config generation.

- `cluster_endpoint` - (Required, String, ForceNew) Kubernetes API endpoint URL (e.g., `"https://10.10.88.73:6443"`).

- `control_plane` - (Required, Block, ForceNew) Control plane node configuration. At least one control plane is required. See [Node Configuration](#node-configuration) below.

### Optional Arguments

- `talos_version` - (Optional, String) Talos version for reference (not used in provisioning).

- `kubernetes_version` - (Optional, String) Kubernetes version for reference.

- `install_disk` - (Optional, String, ForceNew) Install disk for Talos. Defaults to `"/dev/mmcblk0"` (eMMC on RK1).

- `worker` - (Optional, Block, ForceNew, Repeatable) Worker node configurations. Can be specified multiple times.

- `allow_scheduling_on_control_plane` - (Optional, Boolean, ForceNew) Allow scheduling workloads on control plane nodes. Defaults to `true`.

- `metallb` - (Optional, Block) MetalLB load balancer configuration. See [MetalLB Configuration](#metallb-configuration) below.

- `ingress` - (Optional, Block) NGINX Ingress controller configuration. See [Ingress Configuration](#ingress-configuration) below.

- `bootstrap_timeout` - (Optional, Integer) Timeout in seconds for cluster bootstrap operations. Defaults to `600` (10 minutes).

- `kubeconfig_path` - (Optional, String) Path to write the kubeconfig file.

- `talosconfig_path` - (Optional, String) Path to write the talosconfig file.

- `secrets_path` - (Optional, String) Path to write the cluster secrets file (for backup/recovery).

### Node Configuration

Each node block (`control_plane` or `worker`) accepts the following arguments:

- `host` - (Required, String) IP address or hostname of the node.

- `hostname` - (Optional, String) Hostname to assign to the node. Defaults to `turing-cp-N` for control planes or `turing-w-N` for workers.

### MetalLB Configuration

The `metallb` block accepts the following arguments:

- `enabled` - (Optional, Boolean) Whether to deploy MetalLB. Defaults to `false`.

- `ip_range` - (Required if enabled, String) The IP address range for MetalLB to allocate (e.g., `"10.10.88.80-10.10.88.89"`).

### Ingress Configuration

The `ingress` block accepts the following arguments:

- `enabled` - (Optional, Boolean) Whether to deploy NGINX Ingress controller. Defaults to `false`.

- `ip` - (Optional, String) The LoadBalancer IP for the Ingress controller. If not specified and MetalLB is enabled, uses the first IP from the MetalLB range.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The resource identifier (same as `name`).

- `kubeconfig` - (Sensitive) The kubeconfig content for accessing the Kubernetes cluster.

- `talosconfig` - (Sensitive) The talosconfig content for talosctl CLI operations.

- `secrets_yaml` - (Sensitive) The cluster secrets (PKI) in YAML format. Store securely for cluster recovery.

- `api_endpoint` - The Kubernetes API server endpoint URL.

- `cluster_status` - The current status of the cluster (`"bootstrapping"`, `"ready"`, `"degraded"`).

## Timeouts

The following timeouts are configurable via the `bootstrap_timeout` argument:

| Operation | Default | Description |
|-----------|---------|-------------|
| Cluster bootstrap | 10 min | Time to wait for etcd and API server |
| Node health check | 10 min | Time to wait for each node |
| MetalLB deployment | 5 min | Time to wait for MetalLB |
| Ingress deployment | 5 min | Time to wait for Ingress controller |

## Prerequisites

### talosctl Installation

The `talosctl` CLI must be installed on the machine running Terraform:

```bash
# macOS
brew install siderolabs/tap/talosctl

# Linux
curl -sL https://talos.dev/install | sh

# Verify installation
talosctl version --client
```

### Talos Image

Nodes must be flashed with a Talos Linux image before using this resource. For Turing RK1 nodes:

1. Create a schematic for Image Factory:
```yaml
# talos-schematic.yaml
overlay:
  name: turingrk1
  image: siderolabs/sbc-rockchip
customization:
  systemExtensions:
    officialExtensions:
      - siderolabs/iscsi-tools
      - siderolabs/util-linux-tools
```

2. Submit to Image Factory and download:
```bash
# Get schematic ID
SCHEMATIC_ID=$(curl -X POST --data-binary @talos-schematic.yaml \
  https://factory.talos.dev/schematics | jq -r '.id')

# Download image
curl -L -o metal-arm64.raw.xz \
  "https://factory.talos.dev/image/${SCHEMATIC_ID}/v1.9.1/metal-arm64.raw.xz"
xz -d metal-arm64.raw.xz
```

3. Flash nodes using `turingpi_node` resource or BMC API.

## Lifecycle

### Create

1. Validates talosctl is available
2. Creates temporary working directory
3. Generates cluster secrets (`talosctl gen secrets`)
4. Generates base machine configs (`talosctl gen config`)
5. Patches configs with hostnames and scheduling options
6. Applies configs to control plane nodes (`talosctl apply-config --insecure`)
7. Bootstraps the cluster (`talosctl bootstrap`)
8. Waits for API server readiness
9. Applies configs to worker nodes
10. Waits for cluster health
11. Retrieves kubeconfig (`talosctl kubeconfig`)
12. Deploys MetalLB if enabled
13. Deploys NGINX Ingress if enabled
14. Writes config files if paths specified

### Read

1. Checks cluster health via talosctl
2. Updates cluster status (ready/degraded)

### Update

Most changes require resource replacement (ForceNew). Only addon configuration (metallb, ingress) can be updated in-place.

### Delete

1. Resets all worker nodes (`talosctl reset`)
2. Resets control plane nodes
3. Removes local config files

## NPU Limitation

Talos Linux uses a mainline kernel which does not include Rockchip NPU (Neural Processing Unit) drivers. The RK3588's 6 TOPS NPU is **not available** when running Talos.

For workloads requiring NPU acceleration:
- Use `turingpi_k3s_cluster` with Armbian (vendor kernel)
- Or run NPU workloads on a separate K3s cluster

## Import

Talos cluster resources cannot be imported as they require secrets that are generated during initial creation.

## Troubleshooting

### talosctl not found

Ensure talosctl is installed and in your PATH:
```bash
which talosctl
talosctl version --client
```

### Bootstrap timeout

Increase the `bootstrap_timeout` value and ensure:
- Nodes are powered on and reachable
- Network connectivity between nodes
- Correct IP addresses in configuration

### Cluster stuck in maintenance mode

Nodes may be waiting for configuration. Check with:
```bash
talosctl --talosconfig ./talosconfig health --nodes <IP>
```

### Reset failed nodes

If destroy fails, manually reset nodes:
```bash
talosctl --talosconfig ./talosconfig reset \
  --nodes <IP> --graceful=false --reboot
```

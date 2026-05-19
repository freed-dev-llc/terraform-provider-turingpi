# Future Modules: K3s and Talos Cluster Deployment

> **STATUS — HISTORICAL.** This document was the original (2025) implementation plan
> for in-provider cluster resources. Both shipped in the v1.1.x line and are now
> **deprecated** (slated for removal in v2.0.0); users should adopt the equivalent
> modules in [`terraform-turingpi-modules`](https://github.com/freed-dev-llc/terraform-turingpi-modules)
> instead. Kept here as a reference for the original design discussion.

This document outlines the implementation plan for two new high-level modules that will enable complete Kubernetes cluster deployment on Turing Pi 2.5 hardware.

## Overview

| Module | Description | Source Repository | NPU Support |
|--------|-------------|-------------------|-------------|
| `turingpi_k3s_cluster` | Deploy K3s cluster with custom Armbian | `turing-ansible-cluster` | ✅ Yes (vendor kernel 6.1.x) |
| `turingpi_talos_cluster` | Deploy Talos-based Kubernetes cluster | `turing-rk1-clust` | ❌ No (mainline kernel) |

---

## Module 1: K3s Cluster (`turingpi_k3s_cluster`)

### Purpose
Deploy a production-ready K3s Kubernetes cluster on Turing Pi RK1 nodes using custom Armbian images with NPU support.

### Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                    Terraform Provider                        │
├─────────────────────────────────────────────────────────────┤
│  turingpi_k3s_cluster resource                              │
│    ├── Phase 1: Flash Armbian images (existing resources)   │
│    ├── Phase 2: Bootstrap OS (Ansible provisioner)          │
│    ├── Phase 3: Install K3s (Ansible provisioner)           │
│    └── Phase 4: Deploy addons (Helm provider)               │
└─────────────────────────────────────────────────────────────┘
```

### Dependencies
- `terraform-provider-turingpi` - BMC operations (flash, power, boot check)
- Ansible - OS bootstrap and K3s installation
- Helm provider - Addon deployment (MetalLB, Longhorn, etc.)

### Proposed Schema

```hcl
resource "turingpi_k3s_cluster" "main" {
  # Cluster Identity
  name = "production"

  # Image Configuration
  armbian_image = {
    path   = "/path/to/armbian.img"  # Local path or URL
    source = "local"                  # local | url | r2
  }

  # Node Configuration
  control_plane = {
    node     = 1                      # TPI slot number
    ip       = "10.10.88.73"
    hostname = "k3s-cp"
    has_nvme = false
  }

  workers = [
    {
      node     = 2
      ip       = "10.10.88.74"
      hostname = "k3s-w1"
      has_nvme = true
    },
    {
      node     = 3
      ip       = "10.10.88.75"
      hostname = "k3s-w2"
      has_nvme = true
    },
    {
      node     = 4
      ip       = "10.10.88.76"
      hostname = "k3s-w3"
      has_nvme = true
    }
  ]

  # K3s Configuration
  k3s_version   = "v1.31.3+k3s1"
  pod_cidr      = "10.244.0.0/16"
  service_cidr  = "10.96.0.0/12"
  cluster_dns   = "10.96.0.10"
  flannel_backend = "vxlan"

  # Networking
  metallb_ip_range = "10.10.88.80-10.10.88.89"
  ingress_ip       = "10.10.88.80"

  # Storage
  longhorn_enabled  = true
  longhorn_replicas = 2
  longhorn_path     = "/var/lib/longhorn"

  # Addons
  addons = {
    metallb         = true
    nginx_ingress   = true
    longhorn        = true
    prometheus      = true
    portainer_agent = false
  }

  # NPU Support
  install_rknn    = true
  rknn_version    = "2.3.2"
  deploy_rkllama  = true

  # Timeouts
  boot_timeout    = 300
  cluster_timeout = 600
}

# Outputs
output "kubeconfig" {
  value     = turingpi_k3s_cluster.main.kubeconfig
  sensitive = true
}

output "cluster_endpoint" {
  value = turingpi_k3s_cluster.main.api_endpoint
}

output "node_status" {
  value = turingpi_k3s_cluster.main.node_status
}
```

### Implementation Approach

**Option A: Ansible Provisioner (Recommended)**
- Leverage existing Ansible roles from `turing-ansible-cluster`
- Use `local-exec` or `remote-exec` provisioners to run playbooks
- Pros: Reuses proven automation, complex logic already implemented
- Cons: Requires Ansible installed, harder to test

**Option B: Native Go Implementation**
- Implement all provisioning logic in Go
- SSH directly to nodes for configuration
- Pros: Pure Terraform, no external dependencies
- Cons: Significant development effort, duplicates Ansible work

**Option C: Hybrid Approach**
- Use Terraform for BMC operations and state management
- Use embedded Ansible runner for OS/K3s configuration
- Expose high-level Terraform interface

### Key Implementation Tasks

1. **Resource Definition** (`resource_k3s_cluster.go`)
   - Define schema with all configuration options
   - Implement CRUD operations
   - Handle cluster lifecycle (create, update, destroy)

2. **Provisioning Logic**
   - Flash nodes using existing `turingpi_node` logic
   - SSH connection management
   - OS bootstrap (packages, kernel modules, sysctl)
   - K3s installation and configuration

3. **Addon Management**
   - Helm chart deployment
   - Wait for readiness conditions
   - Output endpoints and credentials

4. **State Management**
   - Track cluster state (bootstrapping, ready, degraded)
   - Handle partial failures
   - Support cluster recovery

### NPU Support Details

The K3s module includes full NPU support because:
- Armbian uses vendor kernel (6.1.x) with Rockchip drivers
- RKNN runtime can be installed via Ansible
- rkllama service provides LLM inference API

NPU configuration options:
```hcl
npu_config = {
  install_rknn    = true
  rknn_version    = "2.3.2"      # RKNN-Toolkit2 version
  install_rkllm   = true
  rkllm_version   = "1.2.3"      # RKNN-LLM version
  deploy_rkllama  = true         # Deploy rkllama service
  model           = "deepseek"   # Pre-download model
  model_quantization = "w8a8"    # Quantization level
}
```

---

## Module 2: Talos Cluster (`turingpi_talos_cluster`)

### Purpose
Deploy a Talos-based Kubernetes cluster on Turing Pi RK1 nodes using immutable, API-driven Talos Linux.

### Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                    Terraform Provider                        │
├─────────────────────────────────────────────────────────────┤
│  turingpi_talos_cluster resource                            │
│    ├── Phase 1: Generate/fetch Talos image (factory API)    │
│    ├── Phase 2: Flash nodes (existing resources)            │
│    ├── Phase 3: Generate machine configs (talosctl)         │
│    ├── Phase 4: Apply configs & bootstrap (talosctl)        │
│    └── Phase 5: Deploy addons (Helm provider)               │
└─────────────────────────────────────────────────────────────┘
```

### Dependencies
- `terraform-provider-turingpi` - BMC operations
- `talosctl` - Talos configuration and bootstrap
- Helm provider - Addon deployment

### Proposed Schema

```hcl
resource "turingpi_talos_cluster" "main" {
  # Cluster Identity
  name     = "talos-cluster"
  endpoint = "https://10.10.88.73:6443"

  # Image Configuration
  talos_version = "v1.9.0"
  image_source  = "factory"  # factory | local | url

  # Factory Image Configuration (if image_source = "factory")
  image_factory = {
    overlay = "siderolabs/sbc-rockchip"
    extensions = [
      "siderolabs/iscsi-tools",
      "siderolabs/util-linux-tools"
    ]
  }

  # OR Local Image Configuration
  image_path = "/path/to/metal-arm64.raw"  # If image_source = "local"

  # Node Configuration
  control_plane = {
    node     = 1
    ip       = "10.10.88.73"
    hostname = "talos-cp"
    install_disk = "/dev/mmcblk0"
  }

  workers = [
    {
      node         = 2
      ip           = "10.10.88.74"
      hostname     = "talos-w1"
      install_disk = "/dev/mmcblk0"
      nvme_device  = "/dev/nvme0n1"
      nvme_mount   = "/var/lib/longhorn"
    },
    {
      node         = 3
      ip           = "10.10.88.75"
      hostname     = "talos-w2"
      install_disk = "/dev/mmcblk0"
      nvme_device  = "/dev/nvme0n1"
      nvme_mount   = "/var/lib/longhorn"
    },
    {
      node         = 4
      ip           = "10.10.88.76"
      hostname     = "talos-w3"
      install_disk = "/dev/mmcblk0"
      nvme_device  = "/dev/nvme0n1"
      nvme_mount   = "/var/lib/longhorn"
    }
  ]

  # Kubernetes Configuration
  pod_cidr     = "10.244.0.0/16"
  service_cidr = "10.96.0.0/12"

  # Networking
  metallb_ip_range = "10.10.88.80-10.10.88.89"

  # Storage
  longhorn_enabled = true
  longhorn_replicas = 2

  # Addons
  addons = {
    metallb       = true
    nginx_ingress = true
    longhorn      = true
    prometheus    = true
  }

  # Allow workloads on control plane (recommended for 4-node clusters)
  allow_scheduling_on_controlplane = true

  # Timeouts
  flash_timeout     = 1200  # 20 min per node
  bootstrap_timeout = 600
}

# Outputs
output "kubeconfig" {
  value     = turingpi_talos_cluster.main.kubeconfig
  sensitive = true
}

output "talosconfig" {
  value     = turingpi_talos_cluster.main.talosconfig
  sensitive = true
}

output "cluster_endpoint" {
  value = turingpi_talos_cluster.main.api_endpoint
}

output "schematic_id" {
  value = turingpi_talos_cluster.main.schematic_id
}
```

### Implementation Approach

**Using talosctl CLI**
- Shell out to `talosctl` for machine config generation and application
- Use Talos Image Factory API for custom image builds
- Manage secrets and PKI through Terraform state

### Key Implementation Tasks

1. **Talos Image Factory Integration**
   - HTTP client for factory.talos.dev API
   - Submit schematic, get image URL
   - Download and cache images

2. **Machine Config Generation**
   - Generate cluster secrets
   - Generate controlplane.yaml and worker.yaml
   - Apply node-specific patches (hostname, NVMe, etc.)

3. **Cluster Bootstrap**
   - Apply configs to nodes (insecure mode during maintenance)
   - Bootstrap etcd (one-time operation)
   - Wait for cluster health
   - Extract kubeconfig

4. **State Management**
   - Detect if cluster already bootstrapped
   - Handle config updates (rolling upgrades)
   - Support cluster destruction

### NPU Limitation Notice

**⚠️ NPU NOT SUPPORTED with Talos**

Talos uses mainline Linux kernel which lacks Rockchip's proprietary RKNPU driver.

Current status:
- Talos kernel: 6.12.x (mainline)
- Required kernel: 6.1.x (Rockchip vendor)
- NPU driver: Not in mainline, requires proprietary blobs

Workarounds:
1. Use CPU-based inference (slower)
2. Wait for mainline NPU driver (kernel review in progress)
3. Use K3s module instead for NPU workloads

The schema should include a warning:
```hcl
# NOTE: NPU/GPU acceleration NOT available with Talos
# Use turingpi_k3s_cluster for NPU workloads
```

---

## Shared Components

### Common Data Sources

```hcl
# Get BMC info for cluster planning
data "turingpi_info" "bmc" {}

# Check SD card space before flashing
data "turingpi_sdcard" "storage" {}

# Get current power status
data "turingpi_power" "nodes" {}
```

### Shared Helper Functions

Create `provider/cluster_helpers.go`:
```go
// WaitForSSH - Wait for node to accept SSH connections
func WaitForSSH(ip string, timeout time.Duration) error

// WaitForKubeAPI - Wait for Kubernetes API to be ready
func WaitForKubeAPI(endpoint string, kubeconfig []byte, timeout time.Duration) error

// DeployHelmChart - Deploy a Helm chart with values
func DeployHelmChart(kubeconfig []byte, chart HelmChart) error

// ValidateNodeConfig - Validate node configuration
func ValidateNodeConfig(nodes []NodeConfig) error
```

---

## Implementation Phases

### Phase 1: Foundation (v1.1.2)
- [ ] Create cluster helper functions
- [ ] Add Helm chart deployment support
- [ ] Add SSH connection pooling
- [ ] Add kubeconfig management

### Phase 2: K3s Module (v1.1.3)
- [ ] Resource schema definition
- [ ] Node flashing integration
- [ ] SSH-based OS bootstrap
- [ ] K3s installation logic
- [ ] Addon deployment
- [ ] NPU/RKNN setup
- [ ] Comprehensive tests

### Phase 3: Talos Module (v1.1.4)
- [ ] Talos Image Factory client
- [ ] Machine config generation
- [ ] talosctl integration
- [ ] Bootstrap logic
- [ ] Addon deployment
- [ ] Comprehensive tests

### Phase 4: Polish (v1.1.5)
- [ ] Cluster upgrade support
- [ ] Backup/restore capabilities
- [ ] Multi-cluster management
- [ ] Documentation and examples

---

## File Structure

```
terraform-provider-turingpi/
├── provider/
│   ├── cluster_helpers.go          # Shared cluster utilities
│   ├── helm_client.go              # Helm deployment client
│   ├── ssh_client.go               # SSH connection management
│   ├── resource_k3s_cluster.go     # K3s cluster resource
│   ├── resource_k3s_cluster_test.go
│   ├── resource_talos_cluster.go   # Talos cluster resource
│   ├── resource_talos_cluster_test.go
│   └── talos_factory_client.go     # Talos Image Factory API
├── examples/
│   ├── k3s-cluster/
│   │   ├── main.tf
│   │   ├── variables.tf
│   │   └── outputs.tf
│   └── talos-cluster/
│       ├── main.tf
│       ├── variables.tf
│       └── outputs.tf
└── docs/
    ├── resources/
    │   ├── k3s_cluster.md
    │   └── talos_cluster.md
    └── guides/
        ├── k3s-deployment.md
        └── talos-deployment.md
```

---

## References

### Source Repositories
- K3s Ansible: `~/Code/turing-ansible-cluster`
- Talos Configs: `~/Code/turing-rk1-clust`

### External Documentation
- [Talos Image Factory](https://factory.talos.dev)
- [K3s Documentation](https://docs.k3s.io)
- [Longhorn Documentation](https://longhorn.io/docs)
- [MetalLB Documentation](https://metallb.universe.tf)
- [RKNN-Toolkit2](https://github.com/airockchip/rknn-toolkit2)

### Hardware Specifications
- Turing Pi 2.5 BMC API: v2.0.3
- RK3588 SoC: 8-core ARM64, 6 TOPS NPU, Mali-G610 GPU
- Storage: 32GB eMMC + 500GB NVMe per worker

---

## Decision Log

| Date | Decision | Rationale |
|------|----------|-----------|
| 2024-12-29 | Use Ansible provisioner for K3s | Reuse proven automation, complex logic already tested |
| 2024-12-29 | Shell out to talosctl for Talos | Official tool handles PKI/config complexity |
| 2024-12-29 | NPU only supported in K3s module | Talos mainline kernel lacks vendor drivers |
| 2024-12-29 | Phase implementation across releases | Manage complexity, allow feedback between phases |

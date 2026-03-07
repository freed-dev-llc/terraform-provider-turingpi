---
page_title: "turingpi_k3s_cluster Resource - Turing Pi"
subcategory: ""
description: |-
  Deploys a K3s Kubernetes cluster on pre-flashed Turing Pi nodes via SSH.
---

# turingpi_k3s_cluster (Resource)

~> **Deprecation Warning:** This resource is deprecated and will be removed in v2.0.0. Please migrate to the [terraform-turingpi-modules](https://github.com/freed-dev-llc/terraform-turingpi-modules) repository which provides separate, composable modules for cluster deployment and addon management.

Deploys a K3s Kubernetes cluster on pre-flashed Turing Pi nodes. This resource handles K3s server installation on the control plane, agent installation on worker nodes, and optional deployment of MetalLB and NGINX Ingress.

**Note:** Nodes must be pre-flashed with a compatible Linux distribution (e.g., Armbian) and accessible via SSH before using this resource. Use `turingpi_node` to flash firmware if needed.

## Example Usage

### Minimal Single-Node Cluster

```hcl
resource "turingpi_k3s_cluster" "cluster" {
  name = "my-cluster"

  control_plane {
    host     = "10.10.88.73"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }
}

output "kubeconfig" {
  value     = turingpi_k3s_cluster.cluster.kubeconfig
  sensitive = true
}
```

### Multi-Node Cluster with Workers

```hcl
resource "turingpi_k3s_cluster" "cluster" {
  name        = "turing-cluster"
  k3s_version = "v1.31.4+k3s1"

  control_plane {
    host     = "10.10.88.73"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  worker {
    host     = "10.10.88.74"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  worker {
    host     = "10.10.88.75"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  worker {
    host     = "10.10.88.76"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  kubeconfig_path = "./kubeconfig"
}
```

### Full Cluster with MetalLB and Ingress

```hcl
resource "turingpi_k3s_cluster" "production" {
  name        = "production"
  k3s_version = "v1.31.4+k3s1"

  control_plane {
    host     = "10.10.88.73"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  worker {
    host     = "10.10.88.74"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  worker {
    host     = "10.10.88.75"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  worker {
    host     = "10.10.88.76"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

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

  install_timeout = 900
  kubeconfig_path = "./production-kubeconfig"
}

output "api_endpoint" {
  value = turingpi_k3s_cluster.production.api_endpoint
}

output "cluster_status" {
  value = turingpi_k3s_cluster.production.cluster_status
}
```

### Password-Based SSH Authentication

```hcl
resource "turingpi_k3s_cluster" "cluster" {
  name = "my-cluster"

  control_plane {
    host         = "10.10.88.73"
    ssh_user     = "root"
    ssh_password = var.ssh_password
  }

  worker {
    host         = "10.10.88.74"
    ssh_user     = "root"
    ssh_password = var.ssh_password
  }
}

variable "ssh_password" {
  type      = string
  sensitive = true
}
```

## Argument Reference

### Required Arguments

- `name` - (Required, String) The name of the cluster. Used for identification and as part of resource IDs.

- `control_plane` - (Required, Block) Configuration for the control plane node. See [Node Configuration](#node-configuration) below.

### Optional Arguments

- `k3s_version` - (Optional, String) The K3s version to install (e.g., `"v1.31.4+k3s1"`). If not specified, the latest stable version is installed.

- `cluster_token` - (Optional, String, Sensitive) The cluster token for node authentication. If not specified, a random token is generated.

- `worker` - (Optional, Block, Repeatable) Configuration for worker nodes. Can be specified multiple times for multiple workers. See [Node Configuration](#node-configuration) below.

- `pod_cidr` - (Optional, String) The CIDR for pod networking. Defaults to `"10.244.0.0/16"`.

- `service_cidr` - (Optional, String) The CIDR for service networking. Defaults to `"10.96.0.0/12"`.

- `metallb` - (Optional, Block) MetalLB load balancer configuration. See [MetalLB Configuration](#metallb-configuration) below.

- `ingress` - (Optional, Block) NGINX Ingress controller configuration. See [Ingress Configuration](#ingress-configuration) below.

- `install_timeout` - (Optional, Integer) Timeout in seconds for K3s installation operations. Defaults to `600` (10 minutes).

- `kubeconfig_path` - (Optional, String) Path to write the kubeconfig file. If not specified, kubeconfig is only stored in Terraform state.

### Node Configuration

Each node block (`control_plane` or `worker`) accepts the following arguments:

- `host` - (Required, String) The IP address or hostname of the node.

- `ssh_user` - (Required, String) The SSH username for connecting to the node.

- `ssh_key` - (Optional, String, Sensitive) The SSH private key for authentication. Either `ssh_key` or `ssh_password` must be specified.

- `ssh_password` - (Optional, String, Sensitive) The SSH password for authentication. Either `ssh_key` or `ssh_password` must be specified.

- `ssh_port` - (Optional, Integer) The SSH port. Defaults to `22`.

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

- `kubeconfig` - (Sensitive) The kubeconfig content for accessing the cluster.

- `api_endpoint` - The Kubernetes API server endpoint URL (e.g., `https://10.10.88.73:6443`).

- `node_token` - (Sensitive) The node token for joining additional nodes to the cluster.

- `cluster_status` - The current status of the cluster (`"ready"`, `"degraded"`, etc.).

## Timeouts

The following timeouts are configurable via the `install_timeout` argument:

| Operation | Default | Description |
|-----------|---------|-------------|
| K3s server installation | 10 min | Time to wait for K3s server to be ready |
| K3s agent installation | 10 min | Time to wait for each agent to join |
| MetalLB deployment | 5 min | Time to wait for MetalLB to be ready |
| Ingress deployment | 5 min | Time to wait for Ingress controller |

## Prerequisites

Before using this resource, ensure:

1. **Nodes are accessible via SSH** - Each node must be running and reachable on the specified SSH port.

2. **SSH credentials are configured** - Either SSH key or password authentication must be available.

3. **Nodes have a compatible OS** - Armbian or similar Debian-based distribution recommended for RK1 modules.

4. **Network connectivity** - Nodes must be able to communicate with each other and reach the internet for K3s installation.

## Import

K3s cluster resources cannot be imported as they require SSH credentials that are not stored in Terraform state.

## Lifecycle

### Create

1. Validates SSH connectivity to all nodes
2. Generates cluster token if not provided
3. Installs K3s server on control plane
4. Waits for K3s API to be ready
5. Installs K3s agents on worker nodes
6. Waits for all nodes to reach Ready state
7. Deploys MetalLB if enabled
8. Deploys NGINX Ingress if enabled
9. Writes kubeconfig to file if path specified

### Update

Updates to node configuration or K3s version trigger a destroy and recreate of the cluster.

### Delete

1. Uninstalls K3s agents from worker nodes
2. Uninstalls K3s server from control plane
3. Removes kubeconfig file if it was created

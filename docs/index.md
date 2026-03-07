---
page_title: "Turing Pi Provider"
subcategory: ""
description: |-
  Terraform provider for managing Turing Pi 2.5 BMC (Baseboard Management Controller). Use with the terraform-turingpi-modules for cluster deployment.
---

# Turing Pi Provider

The Turing Pi provider enables Terraform-based management of [Turing Pi 2.5](https://turingpi.com/) clusters through the BMC (Baseboard Management Controller) API.

## Features

- **Power Management** - Control power state of individual compute nodes
- **Firmware Flashing** - Flash firmware images to nodes
- **Boot Verification** - Monitor UART output to verify successful boot with configurable patterns
- **Node Provisioning** - Combined resource for complete node management
- **Talos Linux Support** - Built-in support for Talos Linux boot detection
- **TLS Flexibility** - Skip certificate verification for self-signed or expired BMC certificates

## Example Usage

```hcl
terraform {
  required_providers {
    turingpi = {
      source  = "freed-dev-llc/turingpi"
      version = ">= 1.3.0"
    }
  }
}

provider "turingpi" {
  username = "root"
  password = "turing"
  endpoint = "https://turingpi.local"
}

resource "turingpi_power" "node1" {
  node  = 1
  state = true
}
```

### Talos Linux Example

```hcl
resource "turingpi_node" "talos_node" {
  node                 = 1
  power_state          = "on"
  boot_check           = true
  boot_check_pattern   = "machine is running and ready"
  login_prompt_timeout = 180
}
```

## Authentication

The provider requires BMC credentials to authenticate with the Turing Pi board.

### Configuration Options

- `username` - (Required) BMC username. Can also be set via `TURINGPI_USERNAME` environment variable.
- `password` - (Required) BMC password. Can also be set via `TURINGPI_PASSWORD` environment variable.
- `endpoint` - (Optional) BMC API endpoint URL. Defaults to `https://turingpi.local`. Can also be set via `TURINGPI_ENDPOINT` environment variable.
- `insecure` - (Optional) Skip TLS certificate verification. Useful for self-signed or expired certificates. Defaults to `false`. Can also be set via `TURINGPI_INSECURE` environment variable.

### Using Environment Variables

```bash
export TURINGPI_USERNAME=root
export TURINGPI_PASSWORD=turing
export TURINGPI_ENDPOINT=https://192.168.1.100
export TURINGPI_INSECURE=true  # optional, for self-signed/expired certs
```

```hcl
provider "turingpi" {}
```

## Resources

- [turingpi_power](resources/power.md) - Control node power state
- [turingpi_flash](resources/flash.md) - Flash firmware to a node
- [turingpi_node](resources/node.md) - Comprehensive node management

## Related Modules

For cluster deployment, use the [terraform-turingpi-modules](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi):

| Module | Description |
|--------|-------------|
| [flash-nodes](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/flash-nodes) | Flash firmware to multiple nodes |
| [talos-cluster](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/talos-cluster) | Deploy Talos Kubernetes cluster |
| [k3s-cluster](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/k3s-cluster) | Deploy K3s Kubernetes cluster on Armbian |
| [metallb](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/metallb) | MetalLB load balancer addon |
| [ingress-nginx](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/ingress-nginx) | NGINX Ingress controller addon |
| [longhorn](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/longhorn) | Distributed block storage with NVMe support |
| [monitoring](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/monitoring) | Prometheus, Grafana, Alertmanager stack |
| [portainer](https://registry.terraform.io/modules/freed-dev-llc/modules/turingpi/latest/submodules/portainer) | Cluster management agent (CE/BE) |

```hcl
module "flash" {
  source  = "freed-dev-llc/modules/turingpi//modules/flash-nodes"
  version = ">= 1.3.0"

  nodes = {
    1 = { firmware = "talos-arm64.raw" }
    2 = { firmware = "talos-arm64.raw" }
  }
}

module "cluster" {
  source  = "freed-dev-llc/modules/turingpi//modules/talos-cluster"
  version = ">= 1.3.0"

  cluster_name     = "my-cluster"
  cluster_endpoint = "https://192.168.1.101:6443"
  control_plane    = [{ host = "192.168.1.101" }]
  workers          = [{ host = "192.168.1.102" }]
}
```

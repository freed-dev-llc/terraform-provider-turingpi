# Terraform Provider for Turing Pi 2.5

[![Terraform Registry](https://img.shields.io/badge/terraform-registry-blueviolet)](https://registry.terraform.io/providers/freed-dev-llc/turingpi)
[![Go Version](https://img.shields.io/github/go-mod/go-version/freed-dev-llc/terraform-provider-turingpi)](https://go.dev/)
[![Go](https://github.com/freed-dev-llc/terraform-provider-turingpi/actions/workflows/go.yml/badge.svg)](https://github.com/freed-dev-llc/terraform-provider-turingpi/actions/workflows/go.yml)
[![Security](https://github.com/freed-dev-llc/terraform-provider-turingpi/actions/workflows/security.yml/badge.svg)](https://github.com/freed-dev-llc/terraform-provider-turingpi/actions/workflows/security.yml)
[![codecov](https://codecov.io/gh/freed-dev-llc/terraform-provider-turingpi/graph/badge.svg)](https://codecov.io/gh/freed-dev-llc/terraform-provider-turingpi)
[![Release](https://img.shields.io/github/v/release/freed-dev-llc/terraform-provider-turingpi)](https://github.com/freed-dev-llc/terraform-provider-turingpi/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/freed-dev-llc/terraform-provider-turingpi/total)](https://github.com/freed-dev-llc/terraform-provider-turingpi/releases)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

A Terraform provider for managing Turing Pi's Baseboard Management Controller (BMC), enabling power management, firmware flashing, and node provisioning.

## Features

- **Power Management** - Control power state of individual compute nodes (1-4)
- **Firmware Flashing** - Flash firmware images to nodes with automatic resource recreation
- **BMC Firmware Upgrade** - Upgrade BMC firmware with file upload or local file support
- **BMC Reboot & Reload** - Trigger BMC reboot or daemon reload with readiness monitoring
- **UART Access** - Read and write to node serial consoles for boot monitoring and command execution
- **Boot Verification** - Monitor UART output with configurable patterns to verify successful boot
- **USB Routing** - Configure USB routing between nodes and USB-A connector or BMC
- **USB Boot Mode** - Enable USB boot mode for CM4 provisioning and MSD access
- **Network Reset** - Trigger network switch reset for recovery after configuration changes
- **Storage Monitoring** - Query SD card storage capacity and usage
- **Talos Linux Support** - Built-in boot detection for Talos Linux clusters
- **TLS Flexibility** - Skip certificate verification for self-signed or expired BMC certificates
- **Environment Variables** - Configure provider via environment variables for CI/CD pipelines

## Terraform Modules

For cluster deployment, we recommend using the composable [terraform-turingpi-modules](https://github.com/freed-dev-llc/terraform-turingpi-modules) repository:

| Module | Description |
|--------|-------------|
| `flash-nodes` | Flash firmware to Turing Pi nodes |
| `talos-cluster` | Deploy Talos Kubernetes cluster (uses native Talos provider) |
| `k3s-cluster` | Deploy K3s Kubernetes cluster on Armbian via SSH |
| `metallb` | MetalLB load balancer addon |
| `ingress-nginx` | NGINX Ingress controller addon |
| `longhorn` | Distributed block storage with NVMe support |
| `monitoring` | Prometheus, Grafana, Alertmanager stack |
| `portainer` | Cluster management agent (CE/BE) |

### Helper Scripts

The modules repository includes helper scripts for cluster lifecycle management:

| Script | Description |
|--------|-------------|
| [`cluster-preflight.sh`](https://github.com/freed-dev-llc/terraform-turingpi-modules/blob/main/scripts/cluster-preflight.sh) | Pre-deployment validation (tools, BMC, node connectivity) |
| [`talos-wipe.sh`](https://github.com/freed-dev-llc/terraform-turingpi-modules/blob/main/scripts/talos-wipe.sh) | Wipe Talos cluster (NVMe + eMMC) and shutdown nodes |
| [`k3s-wipe.sh`](https://github.com/freed-dev-llc/terraform-turingpi-modules/blob/main/scripts/k3s-wipe.sh) | Wipe K3s cluster (NVMe + eMMC) and shutdown nodes |
| [`find-armbian-image.sh`](https://github.com/freed-dev-llc/terraform-turingpi-modules/blob/main/scripts/find-armbian-image.sh) | Find Armbian images and generate autoconfig files |

> **Note:** The `turingpi_k3s_cluster` and `turingpi_talos_cluster` resources are deprecated and will be removed in v2.0.0. See the [Migration Guide](docs/MIGRATION.md) for upgrade instructions.

## Documentation

- **[Architecture](docs/ARCHITECTURE.md)** - System diagrams, data flows, and component interactions
- **[Migration Guide](docs/MIGRATION.md)** - Migrate from deprecated cluster resources
- **[Terraform Registry](https://registry.terraform.io/providers/freed-dev-llc/turingpi)** - Provider documentation

## Installation

The provider is available on the [Terraform Registry](https://registry.terraform.io/providers/freed-dev-llc/turingpi). Terraform will automatically download it when you run `terraform init`.

## Usage

```hcl
terraform {
  required_providers {
    turingpi = {
      source  = "freed-dev-llc/turingpi"
      version = "~> 1.3.8"
    }
  }
}

provider "turingpi" {
  username = "root"                      # or TURINGPI_USERNAME env var
  password = "turing"                    # or TURINGPI_PASSWORD env var
  endpoint = "https://turingpi.local"    # or TURINGPI_ENDPOINT env var (optional)
  insecure = false                       # or TURINGPI_INSECURE env var (optional)
}
```

Using environment variables:

```bash
export TURINGPI_USERNAME=root
export TURINGPI_PASSWORD=turing
export TURINGPI_ENDPOINT=https://192.168.1.100  # optional
export TURINGPI_INSECURE=true                   # optional, for self-signed/expired certs
```

```hcl
provider "turingpi" {}
```

## Data Sources

### turingpi_info

Retrieve BMC information including version, network, storage, and node power status.

```hcl
data "turingpi_info" "bmc" {}

output "firmware_version" {
  value = data.turingpi_info.bmc.firmware_version
}

output "node_power_status" {
  value = data.turingpi_info.bmc.nodes
}
```

### turingpi_usb

Retrieve current USB routing configuration.

```hcl
data "turingpi_usb" "current" {}

output "usb_config" {
  value = {
    mode  = data.turingpi_usb.current.mode   # "host" or "device"
    node  = data.turingpi_usb.current.node   # 1-4
    route = data.turingpi_usb.current.route  # "usb-a" or "bmc"
  }
}
```

### turingpi_power

Retrieve current power status of all nodes.

```hcl
data "turingpi_power" "status" {}

output "power_status" {
  value = {
    node1      = data.turingpi_power.status.node1  # true/false
    nodes_on   = data.turingpi_power.status.powered_on_count
    nodes_off  = data.turingpi_power.status.powered_off_count
  }
}
```

### turingpi_uart

Read buffered UART (serial console) output from a node. Reading clears the buffer.

```hcl
data "turingpi_uart" "node1" {
  node = 1
}

output "node1_output" {
  value = data.turingpi_uart.node1.output
}
```

### turingpi_sdcard

Retrieve microSD card storage information.

```hcl
data "turingpi_sdcard" "storage" {}

output "sdcard_info" {
  value = {
    total_gb     = data.turingpi_sdcard.storage.total_gb
    free_gb      = data.turingpi_sdcard.storage.free_gb
    used_percent = data.turingpi_sdcard.storage.used_percent
  }
}
```

### turingpi_about

Retrieve BMC daemon version information.

```hcl
data "turingpi_about" "bmc" {}

output "bmc_versions" {
  value = {
    api      = data.turingpi_about.bmc.api_version
    firmware = data.turingpi_about.bmc.firmware_version
    daemon   = data.turingpi_about.bmc.daemon_version
  }
}
```

## Resources

### turingpi_power

Control node power state with on, off, and reset support.

```hcl
resource "turingpi_power" "node1" {
  node  = 1       # Node ID (1-4)
  state = "on"    # "on", "off", or "reset"
}

# Reset (reboot) a node
resource "turingpi_power" "node2_reset" {
  node  = 2
  state = "reset"
}
```

### turingpi_flash

Flash firmware to a node. Changes to `node` or `firmware_file` trigger resource recreation.

```hcl
resource "turingpi_flash" "node1" {
  node          = 1
  firmware_file = "/path/to/firmware.img"
}
```

### turingpi_usb

Manage USB routing between nodes and the USB-A connector or BMC.

```hcl
resource "turingpi_usb" "node1" {
  node  = 1           # Node ID (1-4)
  mode  = "host"      # "host" or "device"
  route = "usb-a"     # "usb-a" or "bmc" (default: "usb-a")
}
```

### turingpi_network_reset

Trigger a network switch reset. Useful for recovering network connectivity after node power changes or firmware updates.

```hcl
resource "turingpi_network_reset" "switch" {
  triggers = {
    # Reset network when node power changes
    node_states = join(",", [for k, v in turingpi_power.nodes : "${k}:${v.current_state}"])
  }
}
```

### turingpi_bmc_firmware

Upgrade the BMC firmware. Supports uploading from Terraform host or using a file on the BMC filesystem.

```hcl
resource "turingpi_bmc_firmware" "upgrade" {
  firmware_file = "/path/to/bmc-firmware.swu"
  timeout       = 300
}

# Or use a file already on the BMC
resource "turingpi_bmc_firmware" "upgrade_local" {
  firmware_file = "/tmp/firmware.swu"
  bmc_local     = true
}
```

### turingpi_uart

Write commands to a node's UART (serial console).

```hcl
resource "turingpi_uart" "node1_cmd" {
  node    = 1
  command = "echo 'Hello from Terraform'\n"
}
```

### turingpi_bmc_reboot

Trigger a BMC reboot with optional readiness wait.

```hcl
resource "turingpi_bmc_reboot" "maintenance" {
  wait_for_ready = true
  ready_timeout  = 120
}
```

### turingpi_bmc_reload

Restart the BMC daemon (softer than full reboot).

```hcl
resource "turingpi_bmc_reload" "daemon" {
  wait_for_ready = true
  ready_timeout  = 30
}
```

### turingpi_usb_boot

Enable USB boot mode for a node (pulls nRPIBOOT pin low for CM4).

```hcl
resource "turingpi_usb_boot" "node1" {
  node = 1
}
```

### turingpi_node_to_msd

Reboot a node into USB Mass Storage Device mode.

```hcl
resource "turingpi_node_to_msd" "node1" {
  node = 1
}
```

### turingpi_clear_usb_boot

Clear USB boot status for a node.

```hcl
resource "turingpi_clear_usb_boot" "node1" {
  node = 1
}
```

### turingpi_node

Comprehensive node management: power control, firmware flashing, and boot verification.

```hcl
resource "turingpi_node" "node1" {
  node                 = 1                              # Node ID (1-4)
  power_state          = "on"                           # "on" or "off" (default: "on")
  firmware_file        = "/path/to/firmware.img"        # optional
  boot_check           = true                           # Monitor UART for boot pattern (default: false)
  boot_check_pattern   = "login:"                       # Pattern to detect (default: "login:")
  login_prompt_timeout = 120                            # Timeout in seconds (default: 60)
}

# For Talos Linux, use the appropriate boot pattern:
resource "turingpi_node" "talos" {
  node               = 2
  boot_check         = true
  boot_check_pattern = "machine is running and ready"
}
```

## Examples

See the [examples](./examples) directory for complete configurations:

- [basic](./examples/basic) - Simple power control
- [flash-firmware](./examples/flash-firmware) - Firmware flashing
- [full-provisioning](./examples/full-provisioning) - Complete node management with boot verification

## Development

Requires [Go 1.23+](https://go.dev/).

```bash
# Clone and build
git clone https://github.com/freed-dev-llc/terraform-provider-turingpi.git
cd terraform-provider-turingpi
go build -o terraform-provider-turingpi

# Run tests
go test -v ./...

# Enable debug logging
export TF_LOG=DEBUG
terraform apply
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

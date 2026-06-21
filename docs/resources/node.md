---
page_title: "turingpi_node Resource - Turing Pi"
subcategory: ""
description: |-
  Comprehensive management of a Turing Pi compute node including power control, firmware flashing, and boot verification.
---

# turingpi_node (Resource)

Provides comprehensive management of a Turing Pi compute node, combining power control, firmware flashing, and boot verification into a single resource.

~> **Note:** Do not manage the same physical node with both `turingpi_node` and `turingpi_power` (or `turingpi_flash`). They write the same BMC power/flash state with no coordination, so two resources targeting one node will fight each other and report perpetual drift. Use `turingpi_node` for combined provisioning, or the focused `turingpi_power` / `turingpi_flash` resources for the same node - not both.

## Example Usage

### Basic Power Control

```hcl
resource "turingpi_node" "node1" {
  node        = 1
  power_state = "on"
}
```

### With Firmware Flashing

```hcl
resource "turingpi_node" "node1" {
  node         = 1
  power_state  = "on"
  firmware_url = "https://example.com/images/armbian.img.xz"
}
```

### Full Provisioning with Boot Verification

```hcl
resource "turingpi_node" "node1" {
  node                 = 1
  power_state          = "on"
  firmware_url         = "https://example.com/images/armbian.img.xz"
  boot_check           = true
  login_prompt_timeout = 120
}
```

### Talos Linux Boot Verification

```hcl
resource "turingpi_node" "talos_node" {
  node                 = 1
  power_state          = "on"
  boot_check           = true
  boot_check_pattern   = "machine is running and ready"
  login_prompt_timeout = 180
}
```

### Complete Cluster Setup

```hcl
variable "firmware_url" {
  description = "HTTP(S) URL the BMC pulls the firmware image from"
  type        = string
  default     = ""
}

# Node 1 - Full provisioning
resource "turingpi_node" "node1" {
  node                 = 1
  power_state          = "on"
  firmware_url         = var.firmware_url != "" ? var.firmware_url : null
  boot_check           = true
  login_prompt_timeout = 120
}

# Node 2 - Power on only
resource "turingpi_node" "node2" {
  node        = 2
  power_state = "on"
  boot_check  = true
}

# Node 3 - Keep powered off
resource "turingpi_node" "node3" {
  node        = 3
  power_state = "off"
}

# Node 4 - Quick power on (no boot check)
resource "turingpi_node" "node4" {
  node        = 4
  power_state = "on"
  boot_check  = false
}

output "node_status" {
  value = {
    node1 = turingpi_node.node1.power_state
    node2 = turingpi_node.node2.power_state
    node3 = turingpi_node.node3.power_state
    node4 = turingpi_node.node4.power_state
  }
}
```

## Argument Reference

- `node` - (Required, Integer) The node ID (1-4). Changing this forces a new resource.
- `power_state` - (Optional, String) The desired power state. Valid values are `"on"` or `"off"`. Defaults to `"on"`.
- `firmware_url` - (Optional, String) HTTP(S) URL the BMC pulls the firmware image from before the power state is applied. This is the reliable flash path: the BMC reports completion only after the eMMC write finishes. Mutually exclusive with `firmware_file`.
- `firmware_file` - (Optional, String, **Deprecated**) Path to a local firmware image streamed to the node. This uses the BMC's streaming-upload path, which is known broken on current BMC firmware (the BMC reports completion before the eMMC write finishes - issue #63), so an apply can succeed against a node that was never actually flashed. Prefer `firmware_url`. Mutually exclusive with `firmware_url`.
- `boot_check` - (Optional, Boolean) Whether to monitor UART output to verify successful boot. Defaults to `false`.
- `boot_check_pattern` - (Optional, String) The pattern to search for in UART output to confirm successful boot. Defaults to `"login:"`. Use `"machine is running and ready"` for Talos Linux.
- `login_prompt_timeout` - (Optional, Integer) Timeout in seconds to wait for boot pattern when `boot_check` is enabled. Defaults to `60`.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - The resource identifier in the format `node-{node}`.

## Boot Verification

When `boot_check` is enabled, the provider monitors the node's UART output for a specific pattern indicating the operating system has successfully booted. This is useful for:

- Ensuring firmware flashing completed successfully
- Waiting for nodes to be ready before dependent operations
- Detecting boot failures

### Supported Operating Systems

| OS | Pattern |
|----|---------|
| Standard Linux | `login:` (default) |
| Talos Linux | `machine is running and ready` |
| Custom | Any string present in UART output |

The `login_prompt_timeout` controls how long to wait for the boot to complete. Increase this value for slower compute modules or complex boot processes.

## Import

This resource does not support `terraform import`.

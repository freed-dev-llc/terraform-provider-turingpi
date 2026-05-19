---
page_title: "turingpi_node_to_msd Resource - Turing Pi"
subcategory: ""
description: |-
  Reboots a node into USB Mass Storage Device (MSD) mode.
---

# turingpi_node_to_msd (Resource)

Reboots a node into USB Mass Storage Device (MSD) mode. This allows the node's storage (eMMC or NVMe) to be accessed as a USB device from another system, enabling direct imaging or file transfer.

This is useful for:
- Flashing OS images directly to compute module storage
- Backing up or restoring node storage
- Accessing node filesystem without network

## Example Usage

### Basic Usage

```hcl
resource "turingpi_node_to_msd" "node1" {
  node = 1
}
```

### MSD Mode for Imaging

```hcl
# First, power off the node
resource "turingpi_power" "node1_off" {
  node  = 1
  state = "off"
}

# Then reboot into MSD mode
resource "turingpi_node_to_msd" "node1" {
  node = 1

  depends_on = [turingpi_power.node1_off]
}

output "msd_ready" {
  value = "Node 1 is now in MSD mode - storage accessible via USB"
}
```

### Triggered MSD Mode

```hcl
variable "image_timestamp" {
  type    = string
  default = ""
}

resource "turingpi_node_to_msd" "node2" {
  node = 2

  triggers = {
    reimage = var.image_timestamp
  }
}
```

### Conditional MSD Mode

```hcl
variable "enter_msd_mode" {
  type    = bool
  default = false
}

resource "turingpi_node_to_msd" "recovery" {
  count = var.enter_msd_mode ? 1 : 0

  node = 1

  triggers = {
    force = timestamp()
  }
}
```

## Argument Reference

- `node` - (Required, Integer) The node number (1-4) to reboot into MSD mode.

- `triggers` - (Optional, Map of String) A map of values that, when changed, will re-trigger MSD mode.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - Set to `node-to-msd-{node}`.
- `last_triggered` - (String) Timestamp (RFC3339 format) when MSD mode was last triggered.

## Behavior Notes

- **Create**: Reboots the specified node into MSD mode.
- **Update**: If `node` or `triggers` change, MSD mode is re-triggered.
- **Read**: This is a trigger resource with no server-side state to read.
- **Delete**: No action is taken; the node remains in its current state.

## Important Considerations

1. **Node Reboot**: This operation reboots the node. Any running workloads will be interrupted.

2. **USB Connection**: After entering MSD mode, the node's storage will be available as a USB device. You need appropriate USB routing configured to access it.

3. **Exit MSD Mode**: To exit MSD mode and boot normally, power cycle the node using `turingpi_power`.

4. **Timing**: It may take several seconds for the node to fully enter MSD mode and for the storage device to appear.

5. **USB Routing**: Ensure USB routing is configured appropriately using `turingpi_usb` to access the MSD device.

## Typical Workflow

```hcl
# 1. Configure USB routing
resource "turingpi_usb" "node1_msd" {
  node  = 1
  mode  = "device"
  route = "bmc"  # or "usb-a"
}

# 2. Enter MSD mode
resource "turingpi_node_to_msd" "node1" {
  node = 1

  depends_on = [turingpi_usb.node1_msd]
}

# 3. Image the node (external process)
# 4. Power cycle to boot normally
resource "turingpi_power" "node1_boot" {
  node  = 1
  state = "on"

  triggers = {
    after_imaging = var.imaging_complete
  }

  depends_on = [turingpi_node_to_msd.node1]
}
```

## API Endpoint Used

| Endpoint | Purpose |
|----------|---------|
| `GET /api/bmc?opt=set&type=node_to_msd&node={n}` | Reboot node into MSD mode |

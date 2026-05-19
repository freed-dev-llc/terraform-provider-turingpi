---
page_title: "turingpi_usb_boot Resource - Turing Pi"
subcategory: ""
description: |-
  Enables USB boot mode for a specified node on the Turing Pi.
---

# turingpi_usb_boot (Resource)

Enables USB boot mode for a specified node. For Raspberry Pi CM4 modules, this pulls pin 93 (nRPIBOOT) low to enable USB boot mode, allowing the node to boot from USB or be accessed as a USB device.

This is useful for:
- Initial provisioning of compute modules via USB
- Accessing eMMC storage for imaging
- Recovery scenarios

When this resource is destroyed, it automatically clears the USB boot status for the node.

## Example Usage

### Basic Usage

```hcl
resource "turingpi_usb_boot" "node1" {
  node = 1
}
```

### Enable USB Boot for Provisioning

```hcl
resource "turingpi_power" "node1_off" {
  node  = 1
  state = "off"
}

resource "turingpi_usb_boot" "node1" {
  node = 1

  depends_on = [turingpi_power.node1_off]
}

# USB boot mode is now enabled - node can be imaged via USB
```

### Triggered USB Boot Mode

```hcl
variable "enable_usb_boot" {
  type    = string
  default = ""
}

resource "turingpi_usb_boot" "node2" {
  node = 2

  triggers = {
    enable = var.enable_usb_boot
  }
}
```

### Multiple Nodes

```hcl
resource "turingpi_usb_boot" "all_nodes" {
  for_each = toset(["1", "2", "3", "4"])

  node = tonumber(each.key)
}
```

## Argument Reference

- `node` - (Required, Integer) The node number (1-4) to enable USB boot mode for.

- `triggers` - (Optional, Map of String) A map of values that, when changed, will re-enable USB boot mode.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - Set to `usb-boot-node-{node}`.
- `last_enabled` - (String) Timestamp (RFC3339 format) when USB boot mode was last enabled.

## Behavior Notes

- **Create**: Enables USB boot mode for the specified node.
- **Update**: If `node` or `triggers` change, USB boot mode is re-enabled.
- **Read**: This is a trigger resource with no server-side state to read.
- **Delete**: Clears USB boot status for the node by calling `clear_usb_boot`.

## Important Considerations

1. **Power State**: The node should typically be powered off before enabling USB boot mode.

2. **CM4 Specific**: This feature is primarily designed for Raspberry Pi CM4 modules. Behavior may vary for other compute modules.

3. **Automatic Cleanup**: When this resource is destroyed, it automatically clears the USB boot status.

4. **Use with turingpi_clear_usb_boot**: If you need to explicitly clear USB boot without destroying the resource, use the `turingpi_clear_usb_boot` resource instead.

## API Endpoint Used

| Endpoint | Purpose |
|----------|---------|
| `GET /api/bmc?opt=set&type=usb_boot&node={n}` | Enable USB boot mode |
| `GET /api/bmc?opt=set&type=clear_usb_boot&node={n}` | Clear USB boot (on destroy) |

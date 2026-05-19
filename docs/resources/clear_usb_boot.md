---
page_title: "turingpi_clear_usb_boot Resource - Turing Pi"
subcategory: ""
description: |-
  Clears USB boot status for a specified node on the Turing Pi.
---

# turingpi_clear_usb_boot (Resource)

Clears USB boot status for a specified node. This resets the USB boot mode configuration that was previously enabled, returning the node to normal boot behavior.

Use this resource when you need to explicitly clear USB boot status without destroying a `turingpi_usb_boot` resource.

## Example Usage

### Basic Usage

```hcl
resource "turingpi_clear_usb_boot" "node1" {
  node = 1
}
```

### Clear After Provisioning

```hcl
# Enable USB boot for provisioning
resource "turingpi_usb_boot" "node1" {
  node = 1
}

# External provisioning happens here...

# Clear USB boot and return to normal boot
resource "turingpi_clear_usb_boot" "node1" {
  node = 1

  triggers = {
    provisioning_complete = var.provisioning_timestamp
  }

  depends_on = [turingpi_usb_boot.node1]
}

# Power on the node
resource "turingpi_power" "node1" {
  node  = 1
  state = "on"

  depends_on = [turingpi_clear_usb_boot.node1]
}
```

### Triggered Clear

```hcl
variable "clear_usb_boot" {
  type    = string
  default = ""
}

resource "turingpi_clear_usb_boot" "node2" {
  node = 2

  triggers = {
    clear = var.clear_usb_boot
  }
}
```

### Clear All Nodes

```hcl
resource "turingpi_clear_usb_boot" "all_nodes" {
  for_each = toset(["1", "2", "3", "4"])

  node = tonumber(each.key)
}
```

## Argument Reference

- `node` - (Required, Integer) The node number (1-4) to clear USB boot status for.

- `triggers` - (Optional, Map of String) A map of values that, when changed, will re-clear USB boot status.

## Attribute Reference

In addition to all arguments above, the following attributes are exported:

- `id` - Set to `clear-usb-boot-node-{node}`.
- `last_cleared` - (String) Timestamp (RFC3339 format) when USB boot status was last cleared.

## Behavior Notes

- **Create**: Clears USB boot status for the specified node.
- **Update**: If `node` or `triggers` change, USB boot status is re-cleared.
- **Read**: This is a trigger resource with no server-side state to read.
- **Delete**: No action is taken.

## Important Considerations

1. **Idempotent**: Clearing USB boot status when it's not set has no adverse effects.

2. **Use Case**: This resource is useful when you want to manage USB boot enable/clear as separate operations, rather than relying on `turingpi_usb_boot` destroy behavior.

3. **Node State**: Clearing USB boot status does not affect the node's current power state.

## API Endpoint Used

| Endpoint | Purpose |
|----------|---------|
| `GET /api/bmc?opt=set&type=clear_usb_boot&node={n}` | Clear USB boot status |

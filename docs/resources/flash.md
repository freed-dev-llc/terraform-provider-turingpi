---
page_title: "turingpi_flash Resource - Turing Pi"
subcategory: ""
description: |-
  Flashes firmware to a Turing Pi compute node.
---

# turingpi_flash (Resource)

Flashes firmware to a Turing Pi compute node. The node is powered off before flashing. Exactly one of `firmware_url` (recommended) or `firmware_file` must be set. Changing `node`, `firmware_url`, or `firmware_file` triggers re-flash.

~> **Note:** Flashing firmware is a destructive operation. Ensure you have the correct firmware file for your compute module.

!> **`firmware_file` is known broken on current BMC firmware** (issue [#63](https://github.com/freed-dev-llc/terraform-provider-turingpi/issues/63)). The BMC reports the flash as `Done` when the multipart upload finishes, **not** when the eMMC write completes — so `terraform apply` can succeed against a node whose new image was never actually written. Use `firmware_url` until [#66](https://github.com/freed-dev-llc/terraform-provider-turingpi/issues/66) lands a fix.

## Example Usage

### URL-based (recommended)

```hcl
resource "turingpi_flash" "node1" {
  node         = 1
  firmware_url = "https://example.com/firmware/turing-rk1-armbian.img.xz"
}
```

The BMC fetches the file directly. Works with any HTTP(S) URL the BMC can reach — Cloudflare R2 / S3 presigned URLs, a local nginx, etc.

### File-based (deprecated, see warning above)

```hcl
resource "turingpi_flash" "node1" {
  node          = 1
  firmware_file = "/path/to/firmware.img"
}
```

## Argument Reference

- `node` — **(Required, Integer, ForceNew)** node ID, 1–4.
- `firmware_url` — **(Optional, String, ForceNew)** HTTP(S) URL the BMC fetches firmware from. Mutually exclusive with `firmware_file`. The BMC waits for the full download + decompress + eMMC write before reporting `Done`, so this is the only path on current firmware that signals completion accurately.
- `firmware_file` — **(Optional, String, ForceNew, Deprecated)** local path to a firmware image, streamed to the BMC. Mutually exclusive with `firmware_url`. See warning above; use `firmware_url` instead.

## Attribute Reference

- `id` — resource identifier, format `flash-node-{node}`.

## Import

Flash resources cannot be imported; they represent a one-time operation.

# Flash Firmware Example

This example shows how to flash firmware to Turing Pi nodes using the BMC's
URL-based pull (added in v1.5.0). The provider's local-file streaming path
(`firmware_file`) is deprecated and known broken — see
`docs/resources/flash.md`.

## Usage

1. Set environment variables:

```bash
export TURINGPI_USERNAME=root
export TURINGPI_PASSWORD=turing
```

2. Initialize and apply with the firmware URL (HTTPS, reachable from the BMC):

```bash
terraform init
terraform plan -var="firmware_url=https://example.local/firmware.img"
terraform apply -var="firmware_url=https://example.local/firmware.img"
```

## Notes

- The `turingpi_flash` resource uses `ForceNew` for both `node` and `firmware_url`
- Changing either value will destroy and recreate the resource (re-flash)
- The URL must be reachable from the BMC's network (typically an internal HTTP
  server or an R2/S3 bucket)

<!-- BEGIN_TF_DOCS -->


## Usage

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_turingpi"></a> [turingpi](#requirement\_turingpi) | >= 1.5.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_turingpi"></a> [turingpi](#provider\_turingpi) | >= 1.5.0 |

## Modules

No modules.

## Resources

| Name | Type |
|------|------|
| [turingpi_flash.node1](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/flash) | resource |
| [turingpi_flash.node2](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/flash) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_firmware_url"></a> [firmware\_url](#input\_firmware\_url) | HTTPS URL the BMC will pull the firmware image from | `string` | n/a | yes |

## Outputs

No outputs.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_firmware_url"></a> [firmware\_url](#input\_firmware\_url) | HTTPS URL the BMC will pull the firmware image from | `string` | n/a | yes |

## Outputs

No outputs.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_turingpi"></a> [turingpi](#provider\_turingpi) | >= 1.5.0 |

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_turingpi"></a> [turingpi](#requirement\_turingpi) | >= 1.5.0 |
<!-- END_TF_DOCS -->

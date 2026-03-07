# Flash Firmware Example

This example shows how to flash firmware to Turing Pi nodes.

## Usage

1. Set environment variables:

```bash
export TURINGPI_USERNAME=root
export TURINGPI_PASSWORD=turing
```

2. Initialize and apply with firmware path:

```bash
terraform init
terraform plan -var="firmware_path=/path/to/firmware.img"
terraform apply -var="firmware_path=/path/to/firmware.img"
```

## Notes

- The `turingpi_flash` resource uses `ForceNew` for both `node` and `firmware_file`
- Changing either value will destroy and recreate the resource (re-flash)
- Ensure the firmware file exists and is accessible

<!-- BEGIN_TF_DOCS -->


## Usage

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_turingpi"></a> [turingpi](#requirement\_turingpi) | >= 1.2.0 |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_turingpi"></a> [turingpi](#provider\_turingpi) | >= 1.2.0 |

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
| <a name="input_firmware_path"></a> [firmware\_path](#input\_firmware\_path) | Path to the firmware image file | `string` | n/a | yes |

## Outputs

No outputs.

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_firmware_path"></a> [firmware\_path](#input\_firmware\_path) | Path to the firmware image file | `string` | n/a | yes |

## Outputs

No outputs.

## Providers

| Name | Version |
|------|---------|
| <a name="provider_turingpi"></a> [turingpi](#provider\_turingpi) | >= 1.2.0 |

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_turingpi"></a> [turingpi](#requirement\_turingpi) | >= 1.2.0 |
<!-- END_TF_DOCS -->

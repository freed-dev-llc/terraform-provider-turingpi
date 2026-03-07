# Full Provisioning Example

This example demonstrates comprehensive node management with the `turingpi_node` resource, including power control, firmware flashing, and boot verification.

## Usage

1. Set environment variables:

```bash
export TURINGPI_USERNAME=root
export TURINGPI_PASSWORD=turing
```

2. Initialize and apply:

```bash
terraform init

# Without firmware flashing
terraform apply

# With firmware flashing
terraform apply -var="firmware_path=/path/to/firmware.img"

# With custom boot timeout
terraform apply -var="boot_timeout=180"
```

## Resources

| Node | Power | Firmware | Boot Check | Description |
|------|-------|----------|------------|-------------|
| 1 | on | optional | yes | Full provisioning with optional firmware |
| 2 | on | no | yes | Power on with boot verification |
| 3 | off | no | no | Kept powered off |
| 4 | on | no | no | Quick power on (no boot check) |

## Boot Verification

When `boot_check = true`, the provider monitors UART output for a login prompt. This ensures the node has fully booted before Terraform continues. The `login_prompt_timeout` controls how long to wait (default: 60 seconds).

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
| [turingpi_node.node1](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/node) | resource |
| [turingpi_node.node2](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/node) | resource |
| [turingpi_node.node3](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/node) | resource |
| [turingpi_node.node4](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/node) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_boot_pattern"></a> [boot\_pattern](#input\_boot\_pattern) | Pattern to detect in UART output for boot verification | `string` | `"login:"` | no |
| <a name="input_boot_timeout"></a> [boot\_timeout](#input\_boot\_timeout) | Timeout in seconds to wait for boot completion | `number` | `120` | no |
| <a name="input_firmware_path"></a> [firmware\_path](#input\_firmware\_path) | Path to the firmware image file | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_node_status"></a> [node\_status](#output\_node\_status) | Power state of all provisioned nodes |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_boot_pattern"></a> [boot\_pattern](#input\_boot\_pattern) | Pattern to detect in UART output for boot verification | `string` | `"login:"` | no |
| <a name="input_boot_timeout"></a> [boot\_timeout](#input\_boot\_timeout) | Timeout in seconds to wait for boot completion | `number` | `120` | no |
| <a name="input_firmware_path"></a> [firmware\_path](#input\_firmware\_path) | Path to the firmware image file | `string` | `""` | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_node_status"></a> [node\_status](#output\_node\_status) | Power state of all provisioned nodes |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_turingpi"></a> [turingpi](#provider\_turingpi) | >= 1.2.0 |

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_turingpi"></a> [turingpi](#requirement\_turingpi) | >= 1.2.0 |
<!-- END_TF_DOCS -->

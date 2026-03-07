# Basic Example

This example shows how to control power state for Turing Pi nodes.

## Usage

1. Set environment variables:

```bash
export TURINGPI_USERNAME=root
export TURINGPI_PASSWORD=turing
export TURINGPI_ENDPOINT=https://turingpi.local  # optional
```

2. Initialize and apply:

```bash
terraform init
terraform plan
terraform apply
```

## Resources

- `turingpi_power.node1` - Powers on node 1
- `turingpi_power.node2` - Powers on node 2

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
| [turingpi_power.node1](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/power) | resource |
| [turingpi_power.node2](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/power) | resource |

## Inputs

No inputs.

## Outputs

No outputs.

## Inputs

No inputs.

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

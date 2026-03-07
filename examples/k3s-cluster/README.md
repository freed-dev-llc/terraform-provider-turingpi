# K3s Cluster Example

This example deploys a K3s Kubernetes cluster on pre-flashed Turing Pi nodes.

## Prerequisites

1. Nodes must be flashed with a compatible Linux distribution (e.g., Armbian)
2. SSH access must be configured (key-based recommended)
3. Network connectivity between nodes

## Usage

1. Update the variables or create a `terraform.tfvars` file:

```hcl
control_plane_ip = "10.10.88.73"
worker_ips       = ["10.10.88.74", "10.10.88.75", "10.10.88.76"]
ssh_user         = "root"
ssh_key_path     = "~/.ssh/id_rsa"
metallb_ip_range = "10.10.88.80-10.10.88.89"
ingress_ip       = "10.10.88.80"
```

2. Initialize and apply:

```bash
terraform init
terraform apply
```

3. Use the generated kubeconfig:

```bash
export KUBECONFIG=$(pwd)/kubeconfig
kubectl get nodes
```

## What Gets Deployed

- K3s server on the control plane node
- K3s agents on all worker nodes
- MetalLB for LoadBalancer service support
- NGINX Ingress controller for HTTP/HTTPS routing

## Outputs

| Output | Description |
|--------|-------------|
| `kubeconfig` | Cluster kubeconfig content (sensitive) |
| `api_endpoint` | Kubernetes API server URL |
| `cluster_status` | Current cluster status |
| `kubeconfig_path` | Local path to kubeconfig file |

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
| [turingpi_k3s_cluster.cluster](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/k3s_cluster) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_control_plane_ip"></a> [control\_plane\_ip](#input\_control\_plane\_ip) | IP address of the control plane node | `string` | `"10.10.88.73"` | no |
| <a name="input_ingress_ip"></a> [ingress\_ip](#input\_ingress\_ip) | IP address for NGINX Ingress controller | `string` | `"10.10.88.80"` | no |
| <a name="input_k3s_version"></a> [k3s\_version](#input\_k3s\_version) | K3s version to install (empty for latest stable) | `string` | `""` | no |
| <a name="input_metallb_ip_range"></a> [metallb\_ip\_range](#input\_metallb\_ip\_range) | IP range for MetalLB load balancer | `string` | `"10.10.88.80-10.10.88.89"` | no |
| <a name="input_ssh_key_path"></a> [ssh\_key\_path](#input\_ssh\_key\_path) | Path to SSH private key file | `string` | `"~/.ssh/id_rsa"` | no |
| <a name="input_ssh_user"></a> [ssh\_user](#input\_ssh\_user) | SSH username for node access | `string` | `"root"` | no |
| <a name="input_worker_ips"></a> [worker\_ips](#input\_worker\_ips) | List of worker node IP addresses | `list(string)` | <pre>[<br/>  "10.10.88.74",<br/>  "10.10.88.75",<br/>  "10.10.88.76"<br/>]</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_api_endpoint"></a> [api\_endpoint](#output\_api\_endpoint) | Kubernetes API server endpoint |
| <a name="output_cluster_status"></a> [cluster\_status](#output\_cluster\_status) | Current status of the cluster |
| <a name="output_kubeconfig"></a> [kubeconfig](#output\_kubeconfig) | Kubeconfig content for accessing the cluster |
| <a name="output_kubeconfig_path"></a> [kubeconfig\_path](#output\_kubeconfig\_path) | Path to the kubeconfig file |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_control_plane_ip"></a> [control\_plane\_ip](#input\_control\_plane\_ip) | IP address of the control plane node | `string` | `"10.10.88.73"` | no |
| <a name="input_ingress_ip"></a> [ingress\_ip](#input\_ingress\_ip) | IP address for NGINX Ingress controller | `string` | `"10.10.88.80"` | no |
| <a name="input_k3s_version"></a> [k3s\_version](#input\_k3s\_version) | K3s version to install (empty for latest stable) | `string` | `""` | no |
| <a name="input_metallb_ip_range"></a> [metallb\_ip\_range](#input\_metallb\_ip\_range) | IP range for MetalLB load balancer | `string` | `"10.10.88.80-10.10.88.89"` | no |
| <a name="input_ssh_key_path"></a> [ssh\_key\_path](#input\_ssh\_key\_path) | Path to SSH private key file | `string` | `"~/.ssh/id_rsa"` | no |
| <a name="input_ssh_user"></a> [ssh\_user](#input\_ssh\_user) | SSH username for node access | `string` | `"root"` | no |
| <a name="input_worker_ips"></a> [worker\_ips](#input\_worker\_ips) | List of worker node IP addresses | `list(string)` | <pre>[<br/>  "10.10.88.74",<br/>  "10.10.88.75",<br/>  "10.10.88.76"<br/>]</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_api_endpoint"></a> [api\_endpoint](#output\_api\_endpoint) | Kubernetes API server endpoint |
| <a name="output_cluster_status"></a> [cluster\_status](#output\_cluster\_status) | Current status of the cluster |
| <a name="output_kubeconfig"></a> [kubeconfig](#output\_kubeconfig) | Kubeconfig content for accessing the cluster |
| <a name="output_kubeconfig_path"></a> [kubeconfig\_path](#output\_kubeconfig\_path) | Path to the kubeconfig file |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_turingpi"></a> [turingpi](#provider\_turingpi) | >= 1.2.0 |

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_turingpi"></a> [turingpi](#requirement\_turingpi) | >= 1.2.0 |
<!-- END_TF_DOCS -->

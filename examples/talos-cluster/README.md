# Talos Cluster Example

This example deploys a Talos Kubernetes cluster on pre-flashed Turing Pi nodes.

## Prerequisites

1. `talosctl` must be installed and in PATH
2. Nodes must be flashed with Talos Linux image
3. Nodes must be powered on and network-accessible

### Installing talosctl

```bash
# macOS
brew install siderolabs/tap/talosctl

# Linux
curl -sL https://talos.dev/install | sh
```

### Preparing Talos Image

For Turing RK1 nodes, create a custom image via Image Factory:

```bash
# Create schematic
cat > talos-schematic.yaml << 'EOF'
overlay:
  name: turingrk1
  image: siderolabs/sbc-rockchip
customization:
  systemExtensions:
    officialExtensions:
      - siderolabs/iscsi-tools
      - siderolabs/util-linux-tools
EOF

# Get schematic ID
SCHEMATIC_ID=$(curl -X POST --data-binary @talos-schematic.yaml \
  https://factory.talos.dev/schematics | jq -r '.id')

# Download image
curl -L -o metal-arm64.raw.xz \
  "https://factory.talos.dev/image/${SCHEMATIC_ID}/v1.9.1/metal-arm64.raw.xz"
xz -d metal-arm64.raw.xz
```

Then flash nodes using the Turing Pi BMC.

## Usage

1. Update the variables or create a `terraform.tfvars` file:

```hcl
control_plane_ip = "10.10.88.73"
worker_ips       = ["10.10.88.74", "10.10.88.75", "10.10.88.76"]
cluster_name     = "my-talos-cluster"
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

4. Or use talosctl:

```bash
export TALOSCONFIG=$(pwd)/talosconfig
talosctl health
talosctl dashboard
```

## What Gets Deployed

- Talos control plane on the first node
- Talos workers on remaining nodes
- MetalLB for LoadBalancer service support
- NGINX Ingress controller for HTTP/HTTPS routing

## Outputs

| Output | Description |
|--------|-------------|
| `kubeconfig` | Cluster kubeconfig content (sensitive) |
| `talosconfig` | Talosctl config content (sensitive) |
| `api_endpoint` | Kubernetes API server URL |
| `cluster_status` | Current cluster status |
| `kubeconfig_path` | Local path to kubeconfig file |
| `talosconfig_path` | Local path to talosconfig file |

## Files Generated

- `kubeconfig` - Kubernetes cluster access
- `talosconfig` - Talos CLI configuration
- `secrets.yaml` - Cluster PKI secrets (keep secure!)

## NPU Limitation

Talos uses a mainline kernel without Rockchip NPU drivers. For NPU workloads, use `turingpi_k3s_cluster` with Armbian instead.

## Troubleshooting

### Cluster not bootstrapping

Check node status:
```bash
talosctl --talosconfig ./talosconfig health --nodes 10.10.88.73
```

### Reset a node

```bash
talosctl --talosconfig ./talosconfig reset \
  --nodes 10.10.88.73 --graceful=false --reboot
```

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
| [turingpi_talos_cluster.cluster](https://registry.terraform.io/providers/freed-dev-llc/turingpi/latest/docs/resources/talos_cluster) | resource |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Name of the Talos cluster | `string` | `"turing-talos"` | no |
| <a name="input_control_plane_ip"></a> [control\_plane\_ip](#input\_control\_plane\_ip) | IP address of the control plane node | `string` | `"10.10.88.73"` | no |
| <a name="input_ingress_ip"></a> [ingress\_ip](#input\_ingress\_ip) | IP address for NGINX Ingress controller | `string` | `"10.10.88.80"` | no |
| <a name="input_metallb_ip_range"></a> [metallb\_ip\_range](#input\_metallb\_ip\_range) | IP range for MetalLB load balancer | `string` | `"10.10.88.80-10.10.88.89"` | no |
| <a name="input_worker_ips"></a> [worker\_ips](#input\_worker\_ips) | List of worker node IP addresses | `list(string)` | <pre>[<br/>  "10.10.88.74",<br/>  "10.10.88.75",<br/>  "10.10.88.76"<br/>]</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_api_endpoint"></a> [api\_endpoint](#output\_api\_endpoint) | Kubernetes API server endpoint |
| <a name="output_cluster_status"></a> [cluster\_status](#output\_cluster\_status) | Current status of the cluster |
| <a name="output_kubeconfig"></a> [kubeconfig](#output\_kubeconfig) | Kubeconfig content for accessing the cluster |
| <a name="output_kubeconfig_path"></a> [kubeconfig\_path](#output\_kubeconfig\_path) | Path to the kubeconfig file |
| <a name="output_talosconfig"></a> [talosconfig](#output\_talosconfig) | Talosconfig content for talosctl CLI |
| <a name="output_talosconfig_path"></a> [talosconfig\_path](#output\_talosconfig\_path) | Path to the talosconfig file |

## Inputs

| Name | Description | Type | Default | Required |
|------|-------------|------|---------|:--------:|
| <a name="input_cluster_name"></a> [cluster\_name](#input\_cluster\_name) | Name of the Talos cluster | `string` | `"turing-talos"` | no |
| <a name="input_control_plane_ip"></a> [control\_plane\_ip](#input\_control\_plane\_ip) | IP address of the control plane node | `string` | `"10.10.88.73"` | no |
| <a name="input_ingress_ip"></a> [ingress\_ip](#input\_ingress\_ip) | IP address for NGINX Ingress controller | `string` | `"10.10.88.80"` | no |
| <a name="input_metallb_ip_range"></a> [metallb\_ip\_range](#input\_metallb\_ip\_range) | IP range for MetalLB load balancer | `string` | `"10.10.88.80-10.10.88.89"` | no |
| <a name="input_worker_ips"></a> [worker\_ips](#input\_worker\_ips) | List of worker node IP addresses | `list(string)` | <pre>[<br/>  "10.10.88.74",<br/>  "10.10.88.75",<br/>  "10.10.88.76"<br/>]</pre> | no |

## Outputs

| Name | Description |
|------|-------------|
| <a name="output_api_endpoint"></a> [api\_endpoint](#output\_api\_endpoint) | Kubernetes API server endpoint |
| <a name="output_cluster_status"></a> [cluster\_status](#output\_cluster\_status) | Current status of the cluster |
| <a name="output_kubeconfig"></a> [kubeconfig](#output\_kubeconfig) | Kubeconfig content for accessing the cluster |
| <a name="output_kubeconfig_path"></a> [kubeconfig\_path](#output\_kubeconfig\_path) | Path to the kubeconfig file |
| <a name="output_talosconfig"></a> [talosconfig](#output\_talosconfig) | Talosconfig content for talosctl CLI |
| <a name="output_talosconfig_path"></a> [talosconfig\_path](#output\_talosconfig\_path) | Path to the talosconfig file |

## Providers

| Name | Version |
|------|---------|
| <a name="provider_turingpi"></a> [turingpi](#provider\_turingpi) | >= 1.2.0 |

## Requirements

| Name | Version |
|------|---------|
| <a name="requirement_turingpi"></a> [turingpi](#requirement\_turingpi) | >= 1.2.0 |
<!-- END_TF_DOCS -->

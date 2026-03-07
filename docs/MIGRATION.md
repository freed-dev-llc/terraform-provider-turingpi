# Migration Guide: Cluster Resources to Terraform Modules

This guide helps you migrate from the deprecated `turingpi_k3s_cluster` and `turingpi_talos_cluster` resources to the new [terraform-turingpi-modules](https://github.com/freed-dev-llc/terraform-turingpi-modules) repository.

## Why Migrate?

The monolithic cluster resources are being replaced with composable Terraform modules that:

- **Use native providers**: The Talos module uses the official [Talos Terraform Provider](https://registry.terraform.io/providers/siderolabs/talos/latest) instead of wrapping `talosctl` commands
- **Better separation of concerns**: Cluster deployment, MetalLB, and Ingress are separate modules you can use independently
- **More flexibility**: Mix and match addons, use different versions, apply custom configurations
- **Easier maintenance**: Each module can be updated independently

## Timeline

| Version | Status |
|---------|--------|
| v1.x | Deprecated cluster resources available with warnings |
| v2.0.0 | Cluster resources removed |

## Migration Steps

### Step 1: Export Current State

Before migrating, export your cluster's sensitive data:

```bash
# For Talos clusters
terraform output -raw kubeconfig > kubeconfig.bak
terraform output -raw talosconfig > talosconfig.bak
terraform output -raw secrets_yaml > secrets.yaml.bak

# For K3s clusters
terraform output -raw kubeconfig > kubeconfig.bak
```

### Step 2: Remove Old Resources from State

Remove the deprecated resource from Terraform state without destroying the actual cluster:

```bash
# For Talos
terraform state rm turingpi_talos_cluster.cluster

# For K3s
terraform state rm turingpi_k3s_cluster.cluster
```

### Step 3: Update Your Terraform Configuration

Replace the deprecated resource with the new modules.

#### Talos Migration

**Before (deprecated):**

```hcl
resource "turingpi_talos_cluster" "cluster" {
  name             = "my-cluster"
  cluster_endpoint = "https://192.168.1.101:6443"

  control_plane {
    host     = "192.168.1.101"
    hostname = "cp1"
  }

  worker {
    host     = "192.168.1.102"
    hostname = "worker1"
  }

  worker {
    host     = "192.168.1.103"
    hostname = "worker2"
  }

  metallb {
    enabled  = true
    ip_range = "192.168.1.200-192.168.1.220"
  }

  ingress {
    enabled = true
    ip      = "192.168.1.200"
  }

  kubeconfig_path = "./kubeconfig"
}
```

**After (new modules):**

```hcl
# Deploy Talos cluster using native provider
module "cluster" {
  source = "freed-dev-llc/talos-cluster/turingpi"

  cluster_name     = "my-cluster"
  cluster_endpoint = "https://192.168.1.101:6443"

  control_plane = [
    { host = "192.168.1.101", hostname = "cp1" }
  ]

  workers = [
    { host = "192.168.1.102", hostname = "worker1" },
    { host = "192.168.1.103", hostname = "worker2" }
  ]

  kubeconfig_path = "./kubeconfig"
}

# Configure providers for addons
provider "helm" {
  kubernetes {
    config_path = module.cluster.kubeconfig_path
  }
}

provider "kubectl" {
  config_path = module.cluster.kubeconfig_path
}

# Deploy MetalLB separately
module "metallb" {
  source     = "freed-dev-llc/metallb/kubernetes"
  depends_on = [module.cluster]

  ip_range = "192.168.1.200-192.168.1.220"
}

# Deploy Ingress-NGINX separately
module "ingress" {
  source          = "freed-dev-llc/ingress-nginx/kubernetes"
  depends_on      = [module.metallb]

  loadbalancer_ip = "192.168.1.200"
}
```

#### K3s Migration

**Before (deprecated):**

```hcl
resource "turingpi_k3s_cluster" "cluster" {
  name = "my-cluster"

  control_plane {
    host     = "192.168.1.101"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  worker {
    host     = "192.168.1.102"
    ssh_user = "root"
    ssh_key  = file("~/.ssh/id_rsa")
  }

  metallb {
    enabled  = true
    ip_range = "192.168.1.200-192.168.1.220"
  }
}
```

**After (new modules):**

For K3s, you'll need to use a community K3s Terraform module or manage the cluster manually. The terraform-turingpi-modules focuses on Talos as the recommended distribution.

### Step 4: Import Existing Cluster (Optional)

If you want to manage an existing cluster with the new modules, you may need to import state. The Talos provider supports importing existing clusters:

```bash
# Import machine secrets (you'll need the secrets from your backup)
terraform import module.cluster.talos_machine_secrets.this <cluster-name>
```

### Step 5: Apply Changes

Run Terraform to verify the configuration:

```bash
terraform init
terraform plan
terraform apply
```

## Key Differences

| Feature | Old Resource | New Modules |
|---------|--------------|-------------|
| Talos operations | Wraps `talosctl` CLI | Native Talos provider |
| Addons | Built into resource | Separate modules |
| Configuration | Single resource | Composable modules |
| State management | Monolithic | Per-module |
| Customization | Limited | Full Helm values support |

## Troubleshooting

### "Resource not found" after state removal

This is expected. The cluster still exists but Terraform no longer manages it. After adding the new modules, run `terraform plan` to see what changes would be applied.

### Addon conflicts

If you're getting errors about existing MetalLB or Ingress resources, you may need to clean up the existing installations first:

```bash
kubectl delete namespace metallb-system
kubectl delete namespace ingress-nginx
```

Then apply your Terraform configuration.

### Provider authentication issues

Ensure your providers are configured correctly:

```hcl
provider "helm" {
  kubernetes {
    config_path = "./kubeconfig"
  }
}

provider "kubectl" {
  config_path = "./kubeconfig"
}
```

## Questions?

- Open an issue: https://github.com/freed-dev-llc/terraform-turingpi-modules/issues
- Review examples: https://github.com/freed-dev-llc/terraform-turingpi-modules/tree/main/examples

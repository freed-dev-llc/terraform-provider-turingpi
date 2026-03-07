terraform {
  required_providers {
    turingpi = {
      source  = "freed-dev-llc/turingpi"
      version = ">= 1.2.0"
    }
  }
}

provider "turingpi" {}

variable "control_plane_ip" {
  description = "IP address of the control plane node"
  type        = string
  default     = "10.10.88.73"
}

variable "worker_ips" {
  description = "List of worker node IP addresses"
  type        = list(string)
  default     = ["10.10.88.74", "10.10.88.75", "10.10.88.76"]
}

variable "ssh_user" {
  description = "SSH username for node access"
  type        = string
  default     = "root"
}

variable "ssh_key_path" {
  description = "Path to SSH private key file"
  type        = string
  default     = "~/.ssh/id_rsa"
}

variable "k3s_version" {
  description = "K3s version to install (empty for latest stable)"
  type        = string
  default     = ""
}

variable "metallb_ip_range" {
  description = "IP range for MetalLB load balancer"
  type        = string
  default     = "10.10.88.80-10.10.88.89"
}

variable "ingress_ip" {
  description = "IP address for NGINX Ingress controller"
  type        = string
  default     = "10.10.88.80"
}

# Deploy K3s cluster on pre-flashed Turing Pi nodes
resource "turingpi_k3s_cluster" "cluster" {
  name        = "turing-k3s"
  k3s_version = var.k3s_version

  # Control plane node
  control_plane {
    host     = var.control_plane_ip
    ssh_user = var.ssh_user
    ssh_key  = file(var.ssh_key_path)
  }

  # Worker nodes
  dynamic "worker" {
    for_each = var.worker_ips
    content {
      host     = worker.value
      ssh_user = var.ssh_user
      ssh_key  = file(var.ssh_key_path)
    }
  }

  # MetalLB for LoadBalancer services
  metallb {
    enabled  = true
    ip_range = var.metallb_ip_range
  }

  # NGINX Ingress controller
  ingress {
    enabled = true
    ip      = var.ingress_ip
  }

  # Installation settings
  install_timeout = 900

  # Write kubeconfig to local file
  kubeconfig_path = "${path.module}/kubeconfig"
}

# Outputs
output "kubeconfig" {
  description = "Kubeconfig content for accessing the cluster"
  value       = turingpi_k3s_cluster.cluster.kubeconfig
  sensitive   = true
}

output "api_endpoint" {
  description = "Kubernetes API server endpoint"
  value       = turingpi_k3s_cluster.cluster.api_endpoint
}

output "cluster_status" {
  description = "Current status of the cluster"
  value       = turingpi_k3s_cluster.cluster.cluster_status
}

output "kubeconfig_path" {
  description = "Path to the kubeconfig file"
  value       = "${path.module}/kubeconfig"
}

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

variable "cluster_name" {
  description = "Name of the Talos cluster"
  type        = string
  default     = "turing-talos"
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

# Deploy Talos cluster on pre-flashed Turing Pi nodes
resource "turingpi_talos_cluster" "cluster" {
  name             = var.cluster_name
  cluster_endpoint = "https://${var.control_plane_ip}:6443"

  # Control plane node
  control_plane {
    host     = var.control_plane_ip
    hostname = "turing-cp1"
  }

  # Worker nodes
  dynamic "worker" {
    for_each = var.worker_ips
    content {
      host     = worker.value
      hostname = "turing-w${worker.key + 1}"
    }
  }

  # Allow scheduling on control plane (useful for small clusters)
  allow_scheduling_on_control_plane = true

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

  # Timeouts
  bootstrap_timeout = 900

  # Write configs to local files
  kubeconfig_path  = "${path.module}/kubeconfig"
  talosconfig_path = "${path.module}/talosconfig"
  secrets_path     = "${path.module}/secrets.yaml"
}

# Outputs
output "kubeconfig" {
  description = "Kubeconfig content for accessing the cluster"
  value       = turingpi_talos_cluster.cluster.kubeconfig
  sensitive   = true
}

output "talosconfig" {
  description = "Talosconfig content for talosctl CLI"
  value       = turingpi_talos_cluster.cluster.talosconfig
  sensitive   = true
}

output "api_endpoint" {
  description = "Kubernetes API server endpoint"
  value       = turingpi_talos_cluster.cluster.api_endpoint
}

output "cluster_status" {
  description = "Current status of the cluster"
  value       = turingpi_talos_cluster.cluster.cluster_status
}

output "kubeconfig_path" {
  description = "Path to the kubeconfig file"
  value       = "${path.module}/kubeconfig"
}

output "talosconfig_path" {
  description = "Path to the talosconfig file"
  value       = "${path.module}/talosconfig"
}

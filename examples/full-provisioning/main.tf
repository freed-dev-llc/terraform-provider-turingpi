terraform {
  required_providers {
    turingpi = {
      source  = "freed-dev-llc/turingpi"
      version = ">= 1.2.0"
    }
  }
}

provider "turingpi" {}

variable "firmware_path" {
  description = "Path to the firmware image file"
  type        = string
  default     = ""
}

variable "boot_timeout" {
  description = "Timeout in seconds to wait for boot completion"
  type        = number
  default     = 120
}

variable "boot_pattern" {
  description = "Pattern to detect in UART output for boot verification"
  type        = string
  default     = "login:" # Use "machine is running and ready" for Talos Linux
}

# Fully provision node 1 with firmware and boot verification
resource "turingpi_node" "node1" {
  node                 = 1
  power_state          = "on"
  firmware_file        = var.firmware_path != "" ? var.firmware_path : null
  boot_check           = true
  boot_check_pattern   = var.boot_pattern
  login_prompt_timeout = var.boot_timeout
}

# Provision node 2 without firmware flash
resource "turingpi_node" "node2" {
  node               = 2
  power_state        = "on"
  boot_check         = true
  boot_check_pattern = var.boot_pattern
}

# Keep node 3 powered off
resource "turingpi_node" "node3" {
  node        = 3
  power_state = "off"
}

# Node 4 - power on without boot check (faster)
resource "turingpi_node" "node4" {
  node        = 4
  power_state = "on"
  boot_check  = false
}

output "node_status" {
  description = "Power state of all provisioned nodes"
  value = {
    node1 = turingpi_node.node1.power_state
    node2 = turingpi_node.node2.power_state
    node3 = turingpi_node.node3.power_state
    node4 = turingpi_node.node4.power_state
  }
}

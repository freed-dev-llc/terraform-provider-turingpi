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
}

# Flash firmware to node 1
# Note: Changing node or firmware_file will recreate the resource
resource "turingpi_flash" "node1" {
  node          = 1
  firmware_file = var.firmware_path
}

# Flash same firmware to node 2
resource "turingpi_flash" "node2" {
  node          = 2
  firmware_file = var.firmware_path
}

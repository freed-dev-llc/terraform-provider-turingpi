terraform {
  required_providers {
    turingpi = {
      source  = "freed-dev-llc/turingpi"
      version = ">= 1.5.0"
    }
  }
}

provider "turingpi" {}

variable "firmware_url" {
  description = "HTTPS URL the BMC will pull the firmware image from"
  type        = string
}

# Flash firmware to node 1 (recommended path — BMC pulls from URL).
# Note: Changing node or firmware_url will recreate the resource.
resource "turingpi_flash" "node1" {
  node         = 1
  firmware_url = var.firmware_url
}

# Flash same firmware to node 2
resource "turingpi_flash" "node2" {
  node         = 2
  firmware_url = var.firmware_url
}

# Legacy alternative: firmware_file (streaming upload) is deprecated and known
# to return "Done" without actually flashing eMMC on current BMC firmware.
# See docs/resources/flash.md. Use firmware_url unless you have a specific
# reason and have verified the result.

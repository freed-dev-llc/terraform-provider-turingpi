terraform {
  required_providers {
    turingpi = {
      source  = "freed-dev-llc/turingpi"
      version = ">= 1.2.0"
    }
  }
}

# Configure the provider with BMC credentials
# Use environment variables for sensitive values:
#   export TURINGPI_USERNAME=root
#   export TURINGPI_PASSWORD=turing
provider "turingpi" {
  # username = "root"                   # or TURINGPI_USERNAME env var
  # password = "turing"                 # or TURINGPI_PASSWORD env var
  # endpoint = "https://turingpi.local" # or TURINGPI_ENDPOINT env var
}

# Power on node 1
resource "turingpi_power" "node1" {
  node  = 1
  state = "on"
}

# Power on node 2
resource "turingpi_power" "node2" {
  node  = 2
  state = "on"
}

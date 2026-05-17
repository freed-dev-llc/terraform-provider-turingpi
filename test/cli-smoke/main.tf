# Minimal config exercised by the cli-smoketest CI job on every PR.
# The job builds the provider binary, points the CLI at it via
# dev_overrides, then runs `init -input=false` + `validate` under
# both terraform and tofu. Both must succeed.
#
# This catches provider-loading regressions in either tool (e.g. a
# manifest schema field added by HashiCorp that OpenTofu hasn't
# implemented yet, or vice versa).
#
# `validate` runs offline; nothing reaches a real BMC.

terraform {
  required_version = ">= 1.5"

  required_providers {
    turingpi = {
      source  = "freed-dev-llc/turingpi"
      version = ">= 1.0"
    }
  }
}

provider "turingpi" {
  endpoint = "https://bmc.invalid"
  username = "smoketest"
  password = "smoketest"
}

data "turingpi_about" "this" {}

data "turingpi_info" "this" {}

resource "turingpi_power" "node1" {
  node  = 1
  state = "on"
}

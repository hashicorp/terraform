required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = string
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  # We do have a required variable, so this should complain.
  inputs = {}
}

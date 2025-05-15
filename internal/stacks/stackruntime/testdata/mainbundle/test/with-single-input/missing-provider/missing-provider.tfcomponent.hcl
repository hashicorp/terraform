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

  # We do actually require a provider here, Validate() should warn us.
  providers = {}

  inputs = {
    input = var.input
  }
}

removed {
  from = component.removed

  source = "../"

  # We do actually require a provider here, Validate() should warn us.
  providers = {}
}

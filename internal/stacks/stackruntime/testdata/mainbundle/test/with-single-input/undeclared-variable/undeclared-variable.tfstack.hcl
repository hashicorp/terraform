required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    # var.input is not defined
    input = var.input
  }
}

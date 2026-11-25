required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

ephemeral_module "out" {
  source = "./out"

  providers = {
    testing = provider.testing.main
  }
}

component "in" {
  for_each = toset([ephemeral_module.out.value])

  source = "./in"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    input = each.key
  }
}

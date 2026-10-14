required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

ephemeral_component "out" {
  source = "./out"

  providers = {
    testing = provider.testing.main
  }
}

component "in" {
  source = "./in"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    input = ephemeral_component.out.value
  }
}

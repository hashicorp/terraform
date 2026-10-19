required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

ephemeral_component "out" {
  source = "./out"

  providers = {
    testing = provider.testing.default
  }
}

component "in" {
  source = "./in"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id    = "in"
    input = ephemeral_component.out.value
  }

  # This is currently invalid, ephemeral components can't be referenced in depends_on
  depends_on = [ephemeral_component.out]
}

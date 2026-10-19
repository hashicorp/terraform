required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

# The upstream component's plan is always incomplete because its only resource is always deferred
component "deferred_upstream" {
  source = "./upstream"

  providers = {
    testing = provider.testing.main
  }
}

ephemeral_component "out" {
  source = "./out"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id = component.deferred_upstream.id
  }
}

component "in" {
  source = "./in"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id    = "deferred"
    input = ephemeral_component.out.value
  }
}

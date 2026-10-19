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
    input = component.deferred_upstream.static
  }
}

provider "testing" "auth" {
  config {
    authentication = ephemeral_component.out.value
  }
}

# This only refers to the provider, but the provider depends on the ephemeral
# component, which depends on the deferred component.
component "in" {
  source = "./in"

  providers = {
    testing = provider.testing.auth
  }
}

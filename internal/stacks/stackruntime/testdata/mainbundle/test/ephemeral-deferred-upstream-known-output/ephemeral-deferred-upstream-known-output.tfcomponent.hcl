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

# The input is known during planning, but it comes from a component that has an
# incomplete plan, so the ephemeral resource in this component must not be opened.
ephemeral_component "out" {
  source = "./out"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id = component.deferred_upstream.static
  }
}

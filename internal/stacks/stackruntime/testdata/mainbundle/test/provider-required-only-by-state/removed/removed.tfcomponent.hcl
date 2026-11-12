required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

removed {
  from = component.self
  source = "../"

  providers = {
    # The module has no reference to the testing provider, but the component
    # being removed still has a testing resource in state that must be destroyed.
    testing = provider.testing.default
  }
}

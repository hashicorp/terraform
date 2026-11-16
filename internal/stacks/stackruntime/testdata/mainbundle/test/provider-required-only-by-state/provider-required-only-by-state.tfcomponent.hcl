required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "self" {
  source = "./"

  providers = {
    # The component's module has no resources and no required_providers entry for
    # the testing provider, so the configuration alone does not require it. The
    # provider is still needed when a previous run left testing resources in state.
    testing = provider.testing.default
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

removed {
  // this is invalid, without a definition of the stack itself we can't remove
  // components from it directly, instead we should removed the whole stack
  from = stack.embedded.component.self
  source = "../"

  providers = {
    testing = provider.testing.default
  }
}

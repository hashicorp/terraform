required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

stack "nested" {
  source = "../for-each-component"

  inputs = {
    input = {}
  }
}

removed {
  from = stack.nested.component.self["foo"]
  source = "../"

  providers = {
    testing = provider.testing.default
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "auth" {
  config {
    authentication = "embedded-stack-${stack.child.result}"
    require_auth   = true
  }
}

stack "child" {
  source = "./child"
}

component "ephemeral_in" {
  source = "./ephemeral-input"

  providers = {
    testing = provider.testing.auth
  }

  inputs = {
    input = stack.child.result
  }
}

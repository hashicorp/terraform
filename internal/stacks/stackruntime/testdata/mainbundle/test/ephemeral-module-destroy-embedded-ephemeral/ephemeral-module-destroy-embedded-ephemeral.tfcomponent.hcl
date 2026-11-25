required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

component "a" {
  source = "./a"

  providers = {
    testing = provider.testing.main
  }
}

stack "child" {
  source = "./child"

  inputs = {
    input = component.a.value
  }
}

provider "testing" "auth" {
  config {
    authentication = stack.child.result
    require_auth   = true
  }
}

component "b" {
  source = "./b"

  providers = {
    testing = provider.testing.auth
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

ephemeral_component "first" {
  source = "./out"

  providers = {
    testing = provider.testing.main
  }
}

provider "testing" "auth" {
  config {
    authentication = ephemeral_component.first.value
    require_auth   = true
  }
}

ephemeral_component "second" {
  source = "./out"

  providers = {
    testing = provider.testing.auth
  }
}

component "in" {
  source = "./in"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id    = "in"
    input = ephemeral_component.second.value
  }
}

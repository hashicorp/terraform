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

ephemeral_component "e" {
  source = "./e"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    input = component.a.value
  }
}

provider "testing" "auth" {
  config {
    authentication = ephemeral_component.e.value
    require_auth   = true
  }
}

component "b" {
  source = "./b"

  providers = {
    testing = provider.testing.auth
  }
}

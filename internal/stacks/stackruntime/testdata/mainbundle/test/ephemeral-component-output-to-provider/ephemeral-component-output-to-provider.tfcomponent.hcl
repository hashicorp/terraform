required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

ephemeral_component "out" {
  source = "./out"

  providers = {
    testing = provider.testing.main
  }
}

provider "testing" "auth" {
  config {
    authentication = ephemeral_component.out.value
    require_auth   = true
  }
}

component "in" {
  source = "./in"

  providers = {
    testing = provider.testing.auth
  }
}

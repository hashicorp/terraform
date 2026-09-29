required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

component "ephemeral_out" {
  source = "./ephemeral-output"

  providers = {
    testing = provider.testing.main
  }
}

provider "testing" "auth" {
  config {
    authentication = component.ephemeral_out.value
    require_auth   = true
  }
}

component "in" {
  source = "./input"

  providers = {
    testing = provider.testing.auth
  }
}

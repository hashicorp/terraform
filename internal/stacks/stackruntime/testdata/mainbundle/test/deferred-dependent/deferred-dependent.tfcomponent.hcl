required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "valid" {
  source = "../with-single-input"

  providers = {
    testing = provider.testing.default
  }
  inputs = {
    id = "valid"
    input = "valid"
  }
}

component "deferred" {
  source = "../deferrable-component"
  providers = {
    testing = provider.testing.default
  }
  inputs = {
    id = "deferred"
    defer = true
  }
  depends_on = [
    component.valid
  ]
}
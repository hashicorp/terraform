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
        input = "resource"
    }
}

// this component must be created after component.valid
// this component must be destroyed before component.valid
component "self" {
  source = "./"
  providers = {
    testing = provider.testing.default
  }
  inputs = {
    id = "dependent"
    requirements = [
      "valid"
    ]
  }
  depends_on = [
    component.valid
  ]
}

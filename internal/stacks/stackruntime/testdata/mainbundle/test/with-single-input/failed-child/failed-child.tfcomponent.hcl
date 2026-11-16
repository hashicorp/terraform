required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "self" {
  source = "../"
  providers = {
    testing = provider.testing.default
  }
  inputs = {
    id = "self"
    input = "value"
  }
}

component "child" {
  source = "../../failed-component"

  providers = {
    testing = provider.testing.default
  }
  inputs = {
    input = "child"
    fail_apply = true // This will cause the component to fail during apply.
  }
  depends_on = [
    component.self
  ]
}

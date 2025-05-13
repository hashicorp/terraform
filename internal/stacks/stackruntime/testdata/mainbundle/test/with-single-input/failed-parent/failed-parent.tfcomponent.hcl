required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "parent" {
  source = "../../failed-component"

  providers = {
    testing = provider.testing.default
  }
  inputs = {
    input = "Hello, world!"
    fail_apply = true // This will cause the component to fail during apply.
  }
}

component "self" {
  source = "../"
  providers = {
    testing = provider.testing.default
  }
  inputs = {
    input = component.parent.value
  }
}

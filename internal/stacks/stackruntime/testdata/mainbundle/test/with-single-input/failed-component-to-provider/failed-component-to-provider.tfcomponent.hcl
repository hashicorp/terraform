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
    fail_apply = true // This will cause the component to fail during apply.
  }
}

provider "testing" "next" {
  config {
    configure_error = component.parent.value
  }
}

component "self" {
  source = "../"
  providers = {
    testing = provider.testing.next
  }
  inputs = {
    input = "Hello, world!"
  }
}

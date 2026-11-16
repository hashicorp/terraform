required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "parent" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = "parent"
    input = "parent"
  }
}

component "child" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = "child"
    input = component.parent.id
  }
}

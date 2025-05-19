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

provider "testing" "dependent" {
  config {
    ignored = component.parent.id
  }
}

component "child" {
  source = "../"

  providers = {
    testing = provider.testing.dependent
  }

  inputs = {
    id = "child"
    input = "child"
  }
}

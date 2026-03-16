required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}


component "parent" {
  source = "./parent"

  providers = {
    testing = provider.testing.default
  }
}

component "child" {
  source = "./child"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    key = component.parent.deleted_id
  }
}

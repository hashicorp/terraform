required_providers {
  testing = {
    source = "hashicorp/testing"
    version = "0.0.1"
 }
}

locals {
  stringName = "through-local-${component.child.bar}"
  listName = ["through-local-${component.child.bar}"]
  mapName = {
    key = "through-local-${component.child.bar}"
  }
}

provider "testing" "this" {}

component "child" {
  source = "./child"

  inputs = {
    name = "aloha"
    list = ["aloha"]
    map = {
      key = "aloha"
    }
  }

  providers = {
    testing = provider.testing.this
  }
}

component "child2" {
  source = "./child"

  inputs = {
    name = local.stringName
    list = local.listName
    map = local.mapName
  }

  providers = {
    testing = provider.testing.this
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "parent" {
  for_each = toset(["a"])

  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = each.key
    input = "parent"
  }
}

component "child" {
  for_each = toset([ for c in component.parent : c.id ])

  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = "child:${each.key}"
    input = "child"
  }
}

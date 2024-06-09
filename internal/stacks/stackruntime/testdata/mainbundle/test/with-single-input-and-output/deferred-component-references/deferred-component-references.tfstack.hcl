required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "known_components" {
  type = set(string)
}

variable "unknown_components" {
  type = set(string)
}

provider "testing" "default" {}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = each.value
  }

  for_each = var.known_components
}

component "children" {
  // This component validates the behaviour of referencing a known component
  // with an unknown key.
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    // each.key is unknown, but we should still get a typed reference to an
    // output here so we can plan using unknown values.
    input = component.self[each.key].id
  }

  for_each = var.unknown_components
}

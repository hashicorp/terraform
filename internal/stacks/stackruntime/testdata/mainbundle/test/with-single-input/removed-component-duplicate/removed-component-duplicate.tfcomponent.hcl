required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = set(string)
}

variable "removed_one" {
  type = set(string)
}

variable "removed_two" {
  type = set(string)
}

component "self" {
  source = "../"

  for_each = var.input

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id   = each.key
    input = each.key
  }
}

removed {
  from = component.self[each.key]

  source = "../"

  for_each = var.removed_one

  providers = {
    testing = provider.testing.default
  }
}

removed {
  from = component.self[each.key]

  source = "../"

  for_each = var.removed_two

  providers = {
    testing = provider.testing.default
  }
}

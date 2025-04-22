required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = map(string)
  default = {}
}

variable "removed" {
  type = map(string)
  default = {}
}

variable "removed-direct" {
  type = set(string)
  default = []
}

stack "simple" {
  for_each = var.input

  source = "../valid"

  inputs = {
    id = each.key
    input = each.value
  }
}

removed {
  for_each = var.removed

  // This removed block targets the stack directly, and just tells it to
  // remove all components in the stack.

  from = stack.simple[each.key]
  source = "../valid"

  inputs = {
    id = each.key
    input = each.value
  }
}

removed {
  for_each = var.removed-direct

  // This removed block removes the component in the specified stack directly.
  // This is okay as long as only a single component in the stack is being
  // removed. If an entire stack is being removed, you should use the other
  // approach.

  from = stack.simple[each.key].component.self
  source = "../"

  providers = {
    testing = provider.testing.default
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "for_each_input" {
  type = map(string)
  default = {}
}

variable "for_each_removed" {
  type = set(string)
  default = []
}

variable "simple_input" {
  type = map(string)
  default = {}
}

variable "simple_removed" {
  type = set(string)
  default = []
}

stack "for_each" {
  source = "../for-each-component"

  inputs = {
    input = var.for_each_input
  }
}

removed {
  for_each = var.for_each_removed

  from = stack.for_each.component.self[each.key]
  source = "../"

  providers = {
    testing = provider.testing.default
  }
}

stack "simple" {
  for_each = var.simple_input

  source = "../valid"

  inputs = {
    id = each.key
    input = each.value
  }
}

removed {
  for_each = var.simple_removed

  from = stack.simple[each.key].component.self
  source = "../"

  providers = {
    testing = provider.testing.default
  }
}

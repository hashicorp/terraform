required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "input" {
  type = map(string)
  default = {}
}

variable "removed" {
  type = map(string)
  default = {}
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

  from = stack.simple[each.key]
  source = "../valid"

  inputs = {
    id = each.key
    input = each.value
  }
}

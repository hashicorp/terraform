required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "input" {
  type = map(map(string))
  default = {}
}

variable "removed" {
  type = map(map(string))
  default = {}
}

stack "embedded" {
  source = "../removed-stack-instance-dynamic"

  for_each = var.input

  inputs = {
    input = each.value
  }
}

removed {
  for_each = var.removed

  from = stack.embedded[each.key].stack.simple[each.value["id"]]
  source = "../valid"

  inputs = {
    id = each.value["id"]
    input = each.value["input"]
  }
}

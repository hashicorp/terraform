required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "stacks" {
  type = map(string)
}

provider "testing" "default" {}

stack "a" {
  source = "../valid"
  for_each = var.stacks

  inputs = {
    id = each.key
    input = each.value
  }
}

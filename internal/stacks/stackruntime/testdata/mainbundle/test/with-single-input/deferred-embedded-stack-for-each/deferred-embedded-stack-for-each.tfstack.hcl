required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "stacks" {
  type = set(string)
}

provider "testing" "default" {}

stack "a" {
  source = "../valid"
  for_each = var.stacks

  inputs = {
    input = each.value
  }
}

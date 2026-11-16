required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "stacks" {
  type = map(set(string))
}

provider "testing" "default" {}

stack "a" {
  source = "../deferred-component-for-each"
  for_each = var.stacks

  inputs = {
    components = each.value
  }
}

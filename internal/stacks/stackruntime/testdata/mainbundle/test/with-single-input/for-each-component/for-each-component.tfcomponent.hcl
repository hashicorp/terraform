
required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = map(string)
}

component "self" {
  for_each = var.input

  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = each.key
    input = each.value
  }
}

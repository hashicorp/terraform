required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "provider_set" {
  type    = set(string)
  default = ["a", "b"]
}

provider "testing" "main" {}

ephemeral_component "out" {
  source   = "./out"
  for_each = var.provider_set

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id    = each.value
  }
}

component "in" {
  source   = "./in"
  for_each = var.provider_set

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id    = each.value
    input = ephemeral_component.out[each.value].value
  }
}

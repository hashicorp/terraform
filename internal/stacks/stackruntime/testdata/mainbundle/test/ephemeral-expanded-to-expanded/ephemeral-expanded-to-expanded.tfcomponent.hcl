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

component "ephemeral_out" {
  source   = "./ephemeral-output"
  for_each = var.provider_set

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id    = each.value
  }
}

component "ephemeral_in" {
  source   = "./ephemeral-input"
  for_each = var.provider_set

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    id    = each.value
    input = component.ephemeral_out[each.value].value
  }
}

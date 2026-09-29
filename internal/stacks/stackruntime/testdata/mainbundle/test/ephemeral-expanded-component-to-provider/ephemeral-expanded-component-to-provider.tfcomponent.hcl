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
}

provider "testing" "auth" {
  for_each = var.provider_set
  config {
    authentication = component.ephemeral_out[each.value].value
    require_auth   = true
  }
}

component "in" {
  source   = "./input"
  for_each = var.provider_set

  providers = {
    testing = provider.testing.auth[each.value]
  }

  inputs = {
    id = each.value
  }
}

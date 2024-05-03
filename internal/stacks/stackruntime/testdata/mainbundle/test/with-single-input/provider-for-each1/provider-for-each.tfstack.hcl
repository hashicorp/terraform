required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "provider_set" {
  type = set(string)
  default = ["a", "b"]
}


variable "input" {
  type = string
}


provider "testing" "configurations" {
  for_each = var.provider_set
}
component "self" {
  source = "../"
  for_each = var.provider_set

  providers = {
    testing = provider.testing.configurations[each.value]
  }

  inputs = {
    input = var.input
  }
}

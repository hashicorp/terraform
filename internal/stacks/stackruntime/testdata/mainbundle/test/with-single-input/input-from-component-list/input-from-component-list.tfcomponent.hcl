required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "components" {
  type = set(string)
}

provider "testing" "default" {}

component "output" {
  source = "../../with-single-output"

  providers = {
    testing = provider.testing.default
  }

  for_each = var.components
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = component.output[each.value].id
  }

  for_each = var.components
}

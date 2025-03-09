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

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = each.value
  }

  for_each = var.components
}

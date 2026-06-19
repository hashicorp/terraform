required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "component_names" {
  type      = set(string)
  default = ["comp1", "comp2"]
}

locals {
  deferredMap = {
    "comp1": false,
    "comp2": true,
  }
}

component "simple_component" {
  source = "./"
  for_each = var.component_names

  inputs = {
    deferred = local.deferredMap[each.key]
  }

  providers = {
    testing = provider.testing.default
  }
}

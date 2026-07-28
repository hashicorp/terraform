required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {
  for_each = var.component_names
  
  config {
    ignored = each.key
  }
}

variable "component_names" {
  type    = set(string)
  default = ["comp1", "comp2"]
}

component "simple_component" {
  source = "./"
  for_each = setsubtract(var.component_names, ["comp2"])

  inputs = {
    name = each.key
  }

  providers = {
    testing = provider.testing.default[each.key]
  }
}

removed {
  source   = "./"

  from = component.simple_component["comp2"]

  providers = {
    testing = provider.testing.default["comp2"]
  }
}

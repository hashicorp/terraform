required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

stack "embedded" {
  source = "./embedded"
  inputs = {
    component_names = var.component_names
  }
}

variable "component_names" {
  type      = set(string)
  default = ["comp1", "comp2"]
}

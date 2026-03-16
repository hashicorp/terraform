required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "inputs" {
  type = map(string)
}

provider "testing" "default" {}

component "self" {
    for_each = var.inputs

    source = "./"

    providers = {
        testing = provider.testing.default
    }

    inputs = {
        input = each.value
    }
}

component "main" {
    source = "../"

    providers = {
        testing = provider.testing.default
    }

    inputs = {
        input = component.self
    }
}

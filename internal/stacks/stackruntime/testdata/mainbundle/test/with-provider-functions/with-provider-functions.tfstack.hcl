required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = string
}

component "self" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id    = "2f9f3b84"
    input = var.input
  }
}

output "value" {
  type = string
  value = component.self.value
}

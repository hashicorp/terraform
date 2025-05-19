required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "id" {
  type     = string
  default  = null
}

provider "testing" "default" {}

component "sensitive" {
  source = "../../sensitive-output"
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id    = var.id
    input = component.sensitive.out
  }
}

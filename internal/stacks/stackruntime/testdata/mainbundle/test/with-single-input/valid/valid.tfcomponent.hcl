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

variable "id" {
  type = string
  default = null
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = var.id
    input = var.input
  }
}

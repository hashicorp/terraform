required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {
  config {
    ignored = var.ephemeral
  }
}

variable "ephemeral" {
  type      = string
  default   = "Hello, world!"
  ephemeral = true
}

variable "input" {
  type = string
}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
    id = "2f9f3b84"
  }
}

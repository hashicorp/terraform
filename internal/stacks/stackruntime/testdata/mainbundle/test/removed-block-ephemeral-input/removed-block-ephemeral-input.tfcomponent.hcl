required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "secret" {
  type      = string
  ephemeral = true
}

provider "testing" "main" {}

component "in" {
  source = "./in"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    secret = var.secret
  }
}

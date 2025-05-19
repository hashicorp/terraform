required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "unknown" {
    type = string
}

component "self" {
  source = "./"
  providers = {
    testing = provider.testing.default
  }
  inputs = {
    id = "self"
  }
}

component "unknown" {
  source = "./"
  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = var.unknown
  }
}

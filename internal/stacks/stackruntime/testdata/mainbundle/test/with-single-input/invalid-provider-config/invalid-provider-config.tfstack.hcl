required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {
  config {
    // The `imaginary` attribute is not valid for the `testing` provider.
    imaginary = "imaginary"
  }
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
  }
}

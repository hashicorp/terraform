required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input_value" {
  type = string
}

component "simple" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    input_value = var.input_value
  }
}
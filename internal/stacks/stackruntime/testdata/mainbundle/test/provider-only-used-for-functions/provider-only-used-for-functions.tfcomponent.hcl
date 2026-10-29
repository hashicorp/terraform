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

  inputs = {
    input = var.input
  }

  providers = {
    # The component's module only uses the testing provider for a function, so
    # the provider is required for its schema and unconfigured client but there
    # is no provider configuration for it.
    testing = provider.testing.default
  }
}

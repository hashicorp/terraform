required_providers {
  testing = {
    source  = "other/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

variable "input" {
  type = string
}

component "self" {
  source = "../"

  providers = {
    # We don't actually specify a provider type for testing in the underlying
    # module. Terraform will assume it's a HashiCorp provider, but it's not.
    # This should cause an error with a reasonable message.
    testing = provider.testing.default
  }

  inputs = {
    input = var.input
  }
}

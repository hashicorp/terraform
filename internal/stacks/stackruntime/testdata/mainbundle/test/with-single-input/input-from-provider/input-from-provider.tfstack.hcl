required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "self" {
  source = "../"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    # Shouldn't be able to reference providers from here.
    input = provider.testing.default
  }
}

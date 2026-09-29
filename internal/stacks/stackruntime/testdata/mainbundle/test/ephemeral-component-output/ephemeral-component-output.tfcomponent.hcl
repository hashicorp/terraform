required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

component "ephemeral_out" {
  source = "./ephemeral-output"

  providers = {
    testing = provider.testing.main
  }
}

component "ephemeral_in" {
  source = "./ephemeral-input"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    input = component.ephemeral_out.value
  }
}

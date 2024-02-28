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
    // This component doesn't exist. We should see an error.
    input = component.output.id
  }
}

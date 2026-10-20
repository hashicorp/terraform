required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

removed {
  from = component.in

  source = "../removed-block-ephemeral-input/in"

  providers = {
    testing = provider.testing.main
  }
}

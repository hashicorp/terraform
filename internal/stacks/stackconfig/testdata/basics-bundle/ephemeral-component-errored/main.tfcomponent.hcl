required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

ephemeral_component "dupe" {
  source = "./ephemeral-output"

  providers = {
    testing = provider.testing.main
  }
}

ephemeral_component "dupe" {
  source = "./ephemeral-output"

  providers = {
    testing = provider.testing.main
  }
}

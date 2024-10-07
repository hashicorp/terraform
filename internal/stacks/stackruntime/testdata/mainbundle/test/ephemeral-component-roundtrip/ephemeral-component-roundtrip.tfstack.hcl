required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "producer" {
  source = "./producer"


  providers = {
    testing = provider.testing.default
  }

  inputs = {}
}

component "consumer" {
  source = "./consumer"

    providers = {
        testing = provider.testing.default
    }

    inputs = {
        ephemeral_input = component.producer.ephemeral_output
    }
}
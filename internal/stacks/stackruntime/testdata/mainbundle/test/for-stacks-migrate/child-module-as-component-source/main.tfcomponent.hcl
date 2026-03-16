required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "self" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
  }
}

component "triage" {
  source = "./child"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    id = "triage"
    input = "triage_input"
  }
}
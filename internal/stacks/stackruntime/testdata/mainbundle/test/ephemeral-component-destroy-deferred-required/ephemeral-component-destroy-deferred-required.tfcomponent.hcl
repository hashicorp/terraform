required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "a_names" {
  type    = set(string)
  default = ["x"]
}

provider "testing" "main" {}

# When the for_each is unknown, the plan for the existing instance of "a" is incomplete (deferred).
component "a" {
  for_each = var.a_names

  source = "./a"

  providers = {
    testing = provider.testing.main
  }
}

ephemeral_component "e" {
  source = "./e"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    input = component.a["x"].value
  }
}

provider "testing" "auth" {
  config {
    authentication = ephemeral_component.e.value
    require_auth   = true
  }
}

component "b" {
  source = "./b"

  providers = {
    testing = provider.testing.auth
  }
}

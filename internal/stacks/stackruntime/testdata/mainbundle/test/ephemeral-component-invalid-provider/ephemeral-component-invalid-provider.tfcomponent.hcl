required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {
  for_each = toset(["a"])
}

ephemeral_component "out" {
  source = "./out"

  providers = {
    testing = provider.testing.main["missing"]
  }
}

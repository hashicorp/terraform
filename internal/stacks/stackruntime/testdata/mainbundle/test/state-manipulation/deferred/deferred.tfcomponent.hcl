required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "ok" {
  source = "./ok"

  providers = {
    testing = provider.testing.default
  }
}

component "deferred" {
  source = "./deferred"

  providers = {
    testing = provider.testing.default
  }
}

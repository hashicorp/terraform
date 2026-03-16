required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "one" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }

  inputs = {
    value = "bar"
  }
}

removed {
  source = "./"
  from = component.two

  providers = {
    testing = provider.testing.default
  }

  lifecycle {
    destroy = false
  }
}
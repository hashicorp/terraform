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

provider "testing" "other" {
  config {
    ignored = component.one.id
  }
}

removed {
  source = "./"
  from = component.two

  providers = {
    testing = provider.testing.other
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "output" {
  source = "../../with-single-output"

  providers = {
    testing = provider.testing.default
  }
}

removed {
  from = component.self

  source = "../"

  providers = {
    testing = provider.testing.default
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {
  config {
    // This provider is going to fail to configure.
    configure_error = "invalid configuration"
  }
}


component "self" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

removed {
  source = "./"
  from = component.one
  
  providers = {
    testing = provider.testing.default
  }

  lifecycle {
      destroy = false
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
required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

ephemeral_module "main" {
  source = "./"

  providers = {
    testing = provider.testing.default
  }
}

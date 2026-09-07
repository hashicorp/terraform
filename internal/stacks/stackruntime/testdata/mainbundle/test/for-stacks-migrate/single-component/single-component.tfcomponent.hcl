required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}

component "single" {
  source = "./child"

  providers = {
    testing = provider.testing.default
  }
}

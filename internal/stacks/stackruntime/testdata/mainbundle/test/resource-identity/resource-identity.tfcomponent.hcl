required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

component "self" {
  source = "./"
  inputs = {
    name = "example"
  }
  
  providers = {
    testing = provider.testing.main
  }
}

trigger "plan-on-main-branch" {
    check = context.branch == "main"
    is_speculative = false
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "default" {}


component "self" {
  source = "./"
  
  inputs = {
    input = "self"
  }

  providers = {
    testing = provider.testing.default
  }
}


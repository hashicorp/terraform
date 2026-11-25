required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}
 
ephemeral_module "credentials" {
  source = "./credentials"
  inputs = {
    id = "credential-keys"
  }

  providers = {
    testing = provider.testing.main
  }
}

provider "testing" "authenticated" {
  config {
    authentication = ephemeral_module.credentials.values["a"]
    require_auth   = true
  }
}

component "consumer" {
  source = "./consumer"

  providers = {
    testing = provider.testing.authenticated
  }
}

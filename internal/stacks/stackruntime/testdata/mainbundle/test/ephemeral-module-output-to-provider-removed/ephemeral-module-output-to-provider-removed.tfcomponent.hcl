required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

ephemeral_module "out" {
  source = "../ephemeral-module-output-to-provider/out"

  providers = {
    testing = provider.testing.main
  }
}

provider "testing" "auth" {
  config {
    authentication = ephemeral_module.out.value
    require_auth   = true
  }
}

removed {
  from = component.in

  source = "../ephemeral-module-output-to-provider/in"

  providers = {
    testing = provider.testing.auth
  }
}

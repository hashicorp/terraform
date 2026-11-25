required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "secret" {
  type      = string
  ephemeral = true
}

provider "testing" "auth" {
  config {
    authentication = var.secret
    require_auth   = true
  }
}

component "b" {
  source = "./b"

  providers = {
    testing = provider.testing.auth
  }
}

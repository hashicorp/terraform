required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

variable "input" {
  type = string
}

provider "testing" "main" {}

ephemeral_module "e" {
  source = "./e"

  providers = {
    testing = provider.testing.main
  }

  inputs = {
    input = var.input
  }
}

output "result" {
  type      = string
  value     = ephemeral_module.e.value
  ephemeral = true
}

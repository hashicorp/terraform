required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

ephemeral_component "out" {
  source = "./out"

  providers = {
    testing = provider.testing.main
  }
}

output "result" {
  type      = string
  value     = ephemeral_component.out.value
  ephemeral = true
}

required_providers {
  testing = {
    source  = "hashicorp/testing"
    version = "0.1.0"
  }
}

provider "testing" "main" {}

component "ephemeral_out" {
  source = "./ephemeral-output"

  providers = {
    testing = provider.testing.main
  }
}

output "result" {
  type      = string
  value     = component.ephemeral_out.value
  ephemeral = true
}

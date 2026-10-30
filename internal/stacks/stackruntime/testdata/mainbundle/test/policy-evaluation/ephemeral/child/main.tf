terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

ephemeral "testing_resource" "example" {}

output "value" {
  value     = ephemeral.testing_resource.example.value
  ephemeral = true
}

terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

ephemeral "testing_ephem_resource" "data" {}

output "ephemeral_output" {
    ephemeral = true
    value = ephemeral.testing_ephem_resource.data.value
}
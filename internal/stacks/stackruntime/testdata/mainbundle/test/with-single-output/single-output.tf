terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

resource "testing_resource" "data" {}

output "id" {
  value = testing_resource.data.id
}

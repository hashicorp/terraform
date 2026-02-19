terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "id" {
  type = string
}

resource "testing_resource" "data" {
  count = 1
  id    = var.id
}

output "id" {
  value = try(testing_resource.data[0].id, null)
}

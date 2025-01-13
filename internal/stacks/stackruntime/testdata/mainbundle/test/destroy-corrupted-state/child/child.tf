terraform {
  required_providers {
    testing = {
      source = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "key" {
  type = string
}

resource "testing_resource" "primary" {
  for_each = {
    (var.key) = "primary"
  }
  id = each.value
}



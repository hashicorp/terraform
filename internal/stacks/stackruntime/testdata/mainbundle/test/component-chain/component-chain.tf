terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}


variable "id" {
  type     = string
  default  = null
  nullable = true # We'll generate an ID if none provided.
}

variable "value" {
  type = string
}

resource "testing_resource" "data" {
  id    = var.id
  value = var.value
}

output "value" {
  value = testing_resource.data.value
}

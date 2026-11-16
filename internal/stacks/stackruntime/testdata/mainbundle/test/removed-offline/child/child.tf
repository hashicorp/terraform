terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "value" {
  type = string
}

resource "testing_resource" "resource" {
  value = var.value
}

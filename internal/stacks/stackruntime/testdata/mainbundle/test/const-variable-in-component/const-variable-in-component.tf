terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "input" {
  type  = string
  const = true
  default = "hello"
}

resource "testing_resource" "data" {
  value = var.input
}

terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "name" {
  type = string
}

resource "testing_resource" "child_resource" {
  value = "hello from child module in ${var.name}"
}

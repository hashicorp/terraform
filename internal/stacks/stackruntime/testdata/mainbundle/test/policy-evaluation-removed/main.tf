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

resource "testing_resource" "parent_resource" {
  value = "hello from the root of ${var.name}"
}

module "child" {
  source = "./child"
  name   = var.name
}

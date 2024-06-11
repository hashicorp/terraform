terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "deferred" {
  type    = bool
  default = false
}

resource "testing_deferred_resource" "resource" {
  id       = "hello"
  value    = "world"
  deferred = var.deferred
}

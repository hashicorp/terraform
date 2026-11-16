terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "deferred" {
  type = bool
}

resource "testing_deferred_resource" "resource" {
  deferred = var.deferred
}

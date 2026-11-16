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

variable "deferred" {
  type = bool
}

resource "testing_resource" "bucket" {
  id    = "bucket-${var.name}"
  value = "shared"
}

resource "testing_deferred_resource" "block" {
  id       = "block-${var.name}"
  value    = "shared"
  deferred = var.deferred
}

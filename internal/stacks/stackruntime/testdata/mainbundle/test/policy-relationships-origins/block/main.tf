terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "bucket_id" {
  type = string
}

resource "testing_deferred_resource" "block" {
  id       = "block"
  value    = var.bucket_id
  deferred = false
}

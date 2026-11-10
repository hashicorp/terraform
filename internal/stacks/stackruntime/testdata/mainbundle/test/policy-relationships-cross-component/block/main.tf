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

# The block's value is the bucket's id.
resource "testing_deferred_resource" "block" {
  id       = "block"
  value    = "bucket"
  deferred = var.deferred
}

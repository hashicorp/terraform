
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

variable "defer" {
  type = bool
}

resource "testing_deferred_resource" "data" {
  id       = var.id
  deferred = var.defer
}

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

module "grand-child" {
  source   = "./grand-child"
  deferred = var.deferred
}

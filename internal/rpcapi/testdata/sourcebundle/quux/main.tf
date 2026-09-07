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

module "child" {
  source   = "./child"
  deferred = var.deferred
}

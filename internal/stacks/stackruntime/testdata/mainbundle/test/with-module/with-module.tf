terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "create" {
  type = bool
  default = true
}

variable "id" {
  type     = string
  default  = null
  nullable = true # We'll generate an ID if none provided.
}

variable "input" {
  type = string
}

resource "testing_resource" "resource" {
  count = var.create ? 1 : 0
}


module "module" {
  source = "./module"

  providers = {
    testing = testing
  }

  id = testing_resource.resource[0].id
  input = var.input
}

resource "testing_resource" "outside" {
  id    = var.id
  value = var.input
}

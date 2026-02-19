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

variable "input" {
  type = string
}

resource "testing_resource" "data" {
  id    = var.id
  value = var.input
}

resource "testing_resource" "another" {
  count = 2
  id    = var.id
  value = var.input
}

module "child_mod" {
  source = "./child"
  input = var.input
  providers = {
    testing = testing
  }
}

module "child_mod2" {
  source = "./child"
  input = var.input
  # provider block not passed in here
}

output "id" {
  value = testing_resource.data.id
}

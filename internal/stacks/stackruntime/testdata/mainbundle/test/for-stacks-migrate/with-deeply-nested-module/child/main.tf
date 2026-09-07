terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "input" {
  type = string
}

module "grand_child_mod" {
  source = "./grand-child"
  input  = var.input
  # provider block not passed in here
}

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

module "child_mod" {
  source = "./child"
  input  = var.input
  # provider block not passed in here
}

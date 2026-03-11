terraform {
  required_providers {
    tfcoremock = {
      source  = "hashicorp/tfcoremock"
      version = "0.1.1"
    }
  }
}

provider "tfcoremock" {}

variable "input" {
  type = string
}

resource "tfcoremock_simple_resource" "resource" {
  string = var.input
}

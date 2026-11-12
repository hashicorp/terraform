terraform {
  required_providers {
    test = {
      source = "${var.namespace}/test"
    }
  }
}

variable "namespace" {
  type    = string
  const   = true
  default = "hashicorp2"
}

resource "test_instance" "example" {}

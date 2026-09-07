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

resource "testing_resource" "grand_child_data" {
  value = var.input
}

resource "testing_resource" "another_grand_child_data" {
  count = 2
  value = var.input
}

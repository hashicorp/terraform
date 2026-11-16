terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

variable "instances" {
  type = number
}

resource "test_resource" "primary" {
  count = var.instances
}

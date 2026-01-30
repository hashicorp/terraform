terraform {
  required_providers {
    testing = {
      source = "hashicorp/testing"
    }
  }
}

variable "static_input" {
  type = string
}

variable "prefix_input" {
  type = string
}

resource "testing_resource" "static" {
  id = var.static_input
  value = var.static_input
}

resource "testing_resource" "computed" {
  id = "${var.prefix_input}-computed"
  value = "${var.prefix_input}-computed"
}

output "simple_result" {
  value = resource.testing_resource.static.value
}

output "computed_result" {
  value = resource.testing_resource.computed.value
}
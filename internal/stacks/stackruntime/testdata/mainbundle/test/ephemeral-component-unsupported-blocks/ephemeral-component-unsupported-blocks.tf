terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "input" {
  type    = string
  default = "hello"
}

module "child" {
  source = "./child"
}

resource "testing_resource" "data" {
  value = var.input
}

check "example" {
  assert {
    condition     = var.input != ""
    error_message = "never fails"
  }
}

action "testing_action" "example" {
  config {}
}

import {
  to = testing_resource.data
  id = "abc"
}

moved {
  from = testing_resource.old
  to   = testing_resource.data
}

removed {
  from = testing_resource.gone

  lifecycle {
    destroy = false
  }
}

output "not_ephemeral" {
  value = var.input
}

terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "foo" {
  type = string
  validation {
    condition     = length(var.foo) > 0
    error_message = "input must not be empty"
  }
}

resource "testing_resource" "main" {
  id    = "test"
  value = var.foo

  lifecycle {
    postcondition {
      condition     = length(self.value) > 0
      error_message = "value must not be empty"
    }
  }
}

check "value_is_baz" {
  assert {
    condition     = testing_resource.main.value == "baz"
    error_message = "value must be 'baz'"
  }
}
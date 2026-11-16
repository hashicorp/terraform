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

output "value" {
  value = provider::testing::echo(var.input)
}

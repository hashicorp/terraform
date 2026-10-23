terraform {
  required_providers {
    testing = {
      source = "hashicorp/testing"
    }
  }
}

variable "key" {
  type = string

  validation {
    condition     = contains(["a", "b"], var.key)
    error_message = "Unexpected credential key."
  }
}

ephemeral "testing_resource" "credential" {
  for_each = toset([var.key])

  lifecycle {
    precondition {
      condition     = each.key == var.key
      error_message = "The ephemeral resource key must match the module input."
    }
  }
}

output "value" {
  value     = provider::testing::echo(ephemeral.testing_resource.credential[var.key].value)
  ephemeral = true
}

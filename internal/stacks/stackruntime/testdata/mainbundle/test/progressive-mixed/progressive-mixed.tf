terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}

variable "input_value" {
  type = string
}

# Resource that will create changes
resource "testing_resource" "data" {
  id    = "progressive-${var.input_value}"
  value = var.input_value
}

# PRE-APPLY OUTPUTS: These should be available immediately (variable-based)
output "input_echo" {
  value = "input-was-${var.input_value}"
}

output "computed_prefix" {
  value = "computed-${upper(var.input_value)}"
}

# POST-APPLY OUTPUTS: These should only be available after apply (resource-based)
output "resource_value" {
  value = testing_resource.data.value
}

output "resource_id" {
  value = testing_resource.data.id
}
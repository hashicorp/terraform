terraform {
  required_providers {
    testing = {
      source  = "hashicorp/testing"
      version = "0.1.0"
    }
  }
}


variable "id" {
  type     = string
  default  = null
  nullable = true # We'll generate an ID if none provided.
}

variable "value" {
  type = string
}

resource "testing_resource" "data" {
  id    = var.id
  value = var.value
}

# PRE-APPLY OUTPUT: Variable-based, should be available immediately
output "input_echo" {
  value = "input-was-${var.value}"
}

# POST-APPLY OUTPUTS: Resource-dependent, should only be available after apply
# Keep the original 'value' output name for the chain to work
output "value" {
  value = testing_resource.data.value
}

output "resource_id" {
  value = testing_resource.data.id
}

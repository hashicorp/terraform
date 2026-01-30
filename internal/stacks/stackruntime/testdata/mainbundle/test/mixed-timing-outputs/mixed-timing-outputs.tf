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

# Resource that creates something with a unique ID to force changes
resource "testing_resource" "data" {
  id    = "mixed-${var.input_value}"
  value = var.input_value
}

# Local that might be available during different phases
locals {
  computed_value = "computed-${var.input_value}"
  timestamp = formatdate("YYYY-MM-DD", timestamp())
}

# Mix of pre-apply and post-apply outputs

# Should NOT be available pre-apply (depends on resource)
output "resource_value" {
  value = testing_resource.data.value
}

# Should NOT be available pre-apply (depends on resource)  
output "resource_id" {
  value = testing_resource.data.id
}

# Might be available during certain phases (depends on local)
output "local_computed" {
  value = local.computed_value
}

# Might have different availability (depends on timestamp function)
output "timestamp_value" {
  value = local.timestamp
}
terraform {
  required_providers {
    testing = {
      source = "hashicorp/testing"
    }
  }
}

variable "known_input" {
  type = string
}

# Create a resource that will have something to apply
resource "testing_resource" "example" {
  id = "test-resource-${var.known_input}"
  value = "resource-value-${var.known_input}"
}

# Output that depends only on input (should be available pre-apply)
output "simple_known_result" {
  value = "known-${var.known_input}"
}

# Output that depends on resource (only available post-apply)
output "resource_dependent_result" {
  value = resource.testing_resource.example.value
}
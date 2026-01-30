# Component with mixed dependency outputs

variable "known_input" {
  type = string
}

# Output that only depends on inputs - should be evaluatable pre-apply
output "simple_known_result" {
  type = string
  value = "${var.known_input}-processed"
}

# Output that depends on resource state - only evaluatable post-apply
# For this test, we'll use a local value to simulate resource dependency
locals {
  simulated_resource_value = "resource-output-${var.known_input}"
}

output "resource_dependent_result" {
  type = string
  value = local.simulated_resource_value
}
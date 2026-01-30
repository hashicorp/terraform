required_providers {
  testing = {
    source = "hashicorp/testing"
  }
}

variable "known_value" {
  type = string
}

# This component has mixed outputs:
# - Some depend only on known inputs (evaluatable in pre-apply)  
# - Some depend on resource outputs (only evaluatable post-apply)
component "mixed_outputs" {
  source = "./mixed-outputs"
  
  inputs = {
    known_input = var.known_value
  }
}

# This output should be available during pre-apply
output "simple_known_output" {
  type = string
  value = component.mixed_outputs.simple_known_result
}

# This output would only be available after apply (depends on resources)
output "resource_dependent_output" {
  type = string
  value = component.mixed_outputs.resource_dependent_result
}
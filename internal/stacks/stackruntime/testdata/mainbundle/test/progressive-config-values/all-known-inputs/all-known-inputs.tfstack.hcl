required_providers {
  testing = {
    source = "hashicorp/testing"
  }
}

variable "static_value" {
  type = string
}

variable "computed_prefix" {
  type = string
}

# This component has outputs that only depend on inputs,
# so they should be resolvable during the pre-apply phase
component "simple_outputs" {
  source = "./simple-outputs"
  
  inputs = {
    static_input = var.static_value
    prefix_input = var.computed_prefix
  }
}

# Export the component outputs as stack outputs
output "simple_output" {
  type = string
  value = component.simple_outputs.simple_result
}

output "computed_output" {
  type = string  
  value = component.simple_outputs.computed_result
}
required_providers {
  testing = {
    source = "hashicorp/testing"
  }
}

variable "resource_count" {
  type = number
}

# This component has outputs that all depend on resource state
# None should be evaluatable during pre-apply phase
component "resource_dependent" {
  source = "./resource-dependent"
  
  inputs = {
    count_input = var.resource_count
  }
}

# All these outputs depend on resource state
output "resource_ids" {
  type = list(string)
  value = component.resource_dependent.resource_ids
}

output "computed_summary" {
  type = string
  value = component.resource_dependent.computed_summary
}
# Component with outputs that all depend on computed values

variable "count_input" {
  type = number
}

# Simulate resource-dependent values with complex computations
# These would not be evaluatable during pre-apply phase in a real scenario
locals {
  # Use functions that would typically require resource state
  resource_ids = [
    for i in range(var.count_input) : "resource-${i}-${uuid()}"
  ]
}

# Output that depends on generated UUIDs (not evaluatable pre-apply)
output "resource_ids" {
  type = list(string)
  value = local.resource_ids
}

# Output that computes from the generated values
output "computed_summary" {
  type = string
  value = "Created ${length(local.resource_ids)} resources: ${join(", ", local.resource_ids)}"
}
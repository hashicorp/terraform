terraform {
  experiments = [output_type_constraints]
}

output "string" {
  type  = string
  value = "Hello"
}

output "object" {
  type  = object({
    name = optional(string, "Ermintrude"),
  })
  value = {}
}

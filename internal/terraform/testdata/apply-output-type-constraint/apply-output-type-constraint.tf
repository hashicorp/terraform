terraform {
  experiments = [output_type_constraints]
}

output "string" {
  type  = string
  value = true
}

output "object_default" {
  type = object({
    name = optional(string, "Ermintrude")
  })
  value = {}
}

output "object_override" {
  type = object({
    name = optional(string, "Ermintrude")
  })
  value = {
    name = "Peppa"
  }
}

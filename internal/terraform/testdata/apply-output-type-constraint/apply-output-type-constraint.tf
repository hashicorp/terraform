output "string" {
  type  = string
  value = true
}

output "object_default" {
  type = object({
    name = optional(string, "Bart")
  })
  value = {}
}

output "object_override" {
  type = object({
    name = optional(string, "Bart")
  })
  value = {
    name = "Lisa"
  }
}

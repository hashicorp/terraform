output "string" {
  type  = string
  value = "Hello"
}

output "object" {
  type = object({
    name = optional(string, "Bart"),
  })
  value = {}
}

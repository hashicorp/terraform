variable "input" {
  default = null
  type = object({
    organization_name = string
  })
}

output "value" {
  value = var.input
}

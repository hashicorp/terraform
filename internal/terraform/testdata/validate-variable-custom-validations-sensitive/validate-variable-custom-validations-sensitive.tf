
variable "input" {
  type = string
  validation {
    condition = length(var.input) > 5
    error_message = "too short"
  }
  sensitive = true
}

output "value" {
  value = var.input
}

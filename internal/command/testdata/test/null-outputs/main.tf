
variable "input" {
  type = number
}

output "output" {
  value = var.input > 5 ? var.input : null
}

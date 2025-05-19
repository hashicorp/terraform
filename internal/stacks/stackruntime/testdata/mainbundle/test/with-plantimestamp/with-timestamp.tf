output "out" {
  value = "module-output-${plantimestamp()}"
}

variable "value" {
  type = string
}

output "input" {
  value = var.value
}

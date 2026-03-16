
variable "name" {
  type = string
}

output "outputted_name" {
  type = string
  value = "outputted-${var.name}"
}

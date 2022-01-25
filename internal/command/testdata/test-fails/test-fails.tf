variable "input" {
  type = string
}

output "foo" {
  value = "foo value ${var.input}"
}

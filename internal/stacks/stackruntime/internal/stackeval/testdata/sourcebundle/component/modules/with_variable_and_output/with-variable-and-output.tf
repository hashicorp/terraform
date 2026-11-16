variable "test" {
  type    = any
  default = null
}

output "test" {
  value = var.test
}

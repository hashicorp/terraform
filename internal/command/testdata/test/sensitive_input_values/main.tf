variable "password" {
  type = string
}

output "password" {
  value = var.password
  sensitive = true
}

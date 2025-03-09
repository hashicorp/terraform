variable "password" {
  sensitive = true
  type = string
}

output "password" {
  value = var.password
  sensitive = true
}

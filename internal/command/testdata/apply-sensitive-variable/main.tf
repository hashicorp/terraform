variable "shadow" {
  type = string
  sensitive = true
}

output "foo" {
  value = var.shadow
  sensitive = true
}

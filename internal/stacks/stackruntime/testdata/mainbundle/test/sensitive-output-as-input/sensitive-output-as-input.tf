variable "secret" {
  type = string
}

output "result" {
  value     = sensitive(upper(var.secret))
  sensitive = true
}

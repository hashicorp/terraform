variable "in" {
  ephemeral = true
}

output "out" {
  ephemeral = true
  value     = var.in
}

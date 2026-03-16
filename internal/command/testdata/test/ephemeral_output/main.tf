variable "foo" {
    ephemeral = true
    type = string
}
output "value" {
  value = var.foo
  ephemeral = true
}

variable "foo" {
    ephemeral = true
    type = string
}
variable "bar" {
    ephemeral = true
    default = null
    type = string
}

resource "test_resource" "bar" {
  write_only = var.bar
}


output "value" {
  value = "Hello, World!"
}

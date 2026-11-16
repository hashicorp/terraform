
variable "input" {
  type = string
}

resource "test_resource" "foo" {
  value = var.input
}

output "value" {
  value = test_resource.foo.value
}

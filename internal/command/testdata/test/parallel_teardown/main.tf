
variable "input" {
  type = string
}

resource "test_resource" "foo" {
  value = var.input
  destroy_wait_seconds = 5
}

output "value" {
  value = test_resource.foo.value
}

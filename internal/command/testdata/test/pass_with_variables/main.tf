variable "input" {
  type = string
}

resource "test_resource" "foo" {
  value = var.input
}

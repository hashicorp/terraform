variable "input" {
  type    = string
  default = "bar"
}

resource "test_resource" "foo" {
  value = var.input
}

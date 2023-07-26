
variable "input" {
  type = string
}

resource "test_resource" "resource" {
    value = var.input
}

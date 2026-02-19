variable "input" {}

resource "test_resource" "resource" {
  value = var.input
}


variable "input" {
  type = string
}

resource "test_resource" "a" {
  value = var.input
}

resource "test_resource" "c" {}

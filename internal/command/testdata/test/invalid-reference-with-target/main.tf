
variable "input" {
  type = string
}

resource "test_resource" "one" {
  value = var.input
}

resource "test_resource" "two" {}
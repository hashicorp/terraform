
variable "input" {
  type = string
}

resource "test_resource" "resource" {
  id = "resource"
  write_only = var.input
}

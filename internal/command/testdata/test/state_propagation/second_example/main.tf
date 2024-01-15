
variable "input" {
  type = string
}

resource "test_resource" "second_module_resource" {
  id = "b6a1d8cb"
  value = var.input
}

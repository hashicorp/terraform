
variable "input" {
  type = string
}

resource "test_resource" "module_resource" {
  id = "df6h8as9"
  value = var.input
}

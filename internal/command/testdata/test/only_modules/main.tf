
variable "input" {
  type = string
}

resource "test_resource" "resource" {
  id = "598318e0"
  value = var.input
}

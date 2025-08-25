
variable "input" {
  type = string
}

resource "test_resource" "resource" {
  value = var.input
}

output "output" {
  value = test_resource.resource.value
}

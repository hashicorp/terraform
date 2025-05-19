variable "input" {
  type = string
}

resource "test_resource" "resource" {
  value = var.input
}

output "id" {
  value = test_resource.resource.id
}

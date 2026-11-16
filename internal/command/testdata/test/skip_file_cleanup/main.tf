variable "id" {
  type = string
}

resource "test_resource" "resource" {
  value = var.id
}

output "id" {
  value = test_resource.resource.id
}

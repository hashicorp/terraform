variable "value" {
  type = string
}

resource "testing_resource" "resource" {
  value = var.value
}

output "id" {
  value = testing_resource.resource.id
}
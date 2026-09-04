
variable "value" {
  type = string
}

resource "test_resource" "resource" {
  value = var.value
}

output "value" {
  value = test_resource.resource.value
}

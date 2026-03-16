variable "id" {
  type = string
}

variable "unused" {
  type = string
  default = "unused"
}

resource "test_resource" "resource" {
  value = var.id
}

output "id" {
  value = test_resource.resource.id
}

output "unused" {
  value = var.unused
}
variable "id" {
  type = string
}

variable "unused" {
  type = bool
  default = false
}

variable "destroy_fail" {
  type    = bool
  default = false
}

resource "test_resource" "resource" {
  value = var.id
  destroy_fail = var.destroy_fail
}

output "id" {
  value = test_resource.resource.id
}

output "unused" {
  value = var.unused
}
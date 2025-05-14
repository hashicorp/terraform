
variable "destroy_fail" {
  type = bool
  default = null
  nullable = true
}

resource "test_resource" "resource" {
  destroy_fail = var.destroy_fail
}

output "destroy_fail" {
  value = test_resource.resource.destroy_fail
}

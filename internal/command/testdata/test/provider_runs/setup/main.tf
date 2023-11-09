variable "resource_directory" {
  type = string
}

resource "test_resource" "foo" {
  value = var.resource_directory
}

output "resource_directory" {
  value = test_resource.foo.value
}

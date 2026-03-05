variable "id" {
  type = string
}

data "test_data_source" "managed_data" {
  id = var.id
}

resource "test_resource" "foo" {
  value = data.test_data_source.managed_data.value
}

output "value" {
  value = test_resource.foo.value
}

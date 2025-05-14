variable "id" {
  type = string
}

data "test_data_source" "resource" {
  id = var.id
}

output "value" {
  value = data.test_data_source.resource.value
}


variable "managed_id" {
  type = string
}

data "test_data_source" "managed_data" {
  id = var.managed_id
}

resource "test_resource" "created" {
  value = data.test_data_source.managed_data.value
}


variable "input" {
  type = string
}

data "test_data_source" "datasource" {
  id = "resource"
  write_only = var.input
}

resource "test_resource" "resource" {
  value = data.test_data_source.datasource.value
  write_only = var.input
}

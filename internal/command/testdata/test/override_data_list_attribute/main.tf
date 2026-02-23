terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

data "test_data_source" "datasource" {
  id = "resource"
}

output "list_value" {
  value = data.test_data_source.datasource.list_value
}

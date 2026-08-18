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

output "nested_list_value" {
  value = data.test_data_source.datasource.nested_list_value
}

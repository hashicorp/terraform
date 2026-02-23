terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

data "test_complex_data_source" "datasource" {
  id = "resource"
}

output "nested_list_value" {
  value = data.test_complex_data_source.datasource.nested_list_value
}

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

output "nested_set_value" {
  value = data.test_complex_data_source.datasource.nested_set_value
}

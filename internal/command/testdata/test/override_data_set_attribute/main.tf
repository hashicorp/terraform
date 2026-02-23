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

output "set_value" {
  value = data.test_complex_data_source.datasource.set_value
}

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

output "map_value" {
  value = data.test_complex_data_source.datasource.map_value
}

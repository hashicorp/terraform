provider "test" {}

override_data {
  target = data.test_data_source.datasource
  values = {
    nested_list_value = "wrong type"
  }
}

run "test_override_data_nested_list_attribute_invalid_type" {
  command = plan
}

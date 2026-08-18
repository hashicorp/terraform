provider "test" {}

override_data {
  target = data.test_data_source.datasource
  values = {
    nested_list_value = {
      name = "shared"
    }
  }
}

run "test_override_data_nested_list_attribute_object" {
  command = plan

  assert {
    condition     = length(data.test_data_source.datasource.nested_list_value) == 0
    error_message = "Expected nested_list_value to be empty when overridden with an object"
  }
}

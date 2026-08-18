provider "test" {}

override_data {
  target = data.test_complex_data_source.datasource
  values = {
    nested_set_value = {
      name = "shared"
    }
  }
}

run "test_override_data_complex_nested_set_attribute_object" {
  command = plan

  assert {
    condition     = length(data.test_complex_data_source.datasource.nested_set_value) == 0
    error_message = "Expected nested_set_value to be empty when overridden with an object"
  }
}

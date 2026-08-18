provider "test" {}

override_data {
  target = data.test_complex_data_source.datasource
  values = {
    nested_list_value = [
      {
        name  = "first"
        value = "one"
      },
      {
        name  = "second"
        value = "two"
      },
    ]
  }
}

run "test_override_data_nested_list_attribute" {
  command = plan

  assert {
    condition     = length(data.test_complex_data_source.datasource.nested_list_value) == 2
    error_message = "Expected nested_list_value to have 2 elements, got ${length(data.test_complex_data_source.datasource.nested_list_value)}"
  }

  assert {
    condition     = data.test_complex_data_source.datasource.nested_list_value[0].name == "first"
    error_message = "Expected first element name to be 'first'"
  }
}

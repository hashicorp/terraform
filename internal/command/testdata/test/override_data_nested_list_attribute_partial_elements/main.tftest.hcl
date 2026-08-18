provider "test" {}

override_data {
  target = data.test_data_source.datasource
  values = {
    nested_list_value = [
      {
        name = "first"
      },
      {
        value = "two"
      },
    ]
  }
}

run "test_override_data_nested_list_attribute_partial_elements" {
  command = plan

  assert {
    condition     = length(data.test_data_source.datasource.nested_list_value) == 2
    error_message = "Expected nested_list_value to have 2 elements, got ${length(data.test_data_source.datasource.nested_list_value)}"
  }

  assert {
    condition     = data.test_data_source.datasource.nested_list_value[0].name == "first"
    error_message = "Expected first element name to be 'first'"
  }

  assert {
    condition     = data.test_data_source.datasource.nested_list_value[0].value != null
    error_message = "Expected first element value to be filled in"
  }

  assert {
    condition     = data.test_data_source.datasource.nested_list_value[1].value == "two"
    error_message = "Expected second element value to be 'two'"
  }

  assert {
    condition     = data.test_data_source.datasource.nested_list_value[1].name != null
    error_message = "Expected second element name to be filled in"
  }
}

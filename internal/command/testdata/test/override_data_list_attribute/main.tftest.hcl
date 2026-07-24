provider "test" {}

override_data {
  target = data.test_data_source.datasource
  values = {
    list_value = [
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

run "test_override_data_list_attribute" {
  command = plan

  assert {
    condition     = length(data.test_data_source.datasource.list_value) == 2
    error_message = "Expected list_value to have 2 elements, got ${length(data.test_data_source.datasource.list_value)}"
  }

  assert {
    condition     = data.test_data_source.datasource.list_value[0].name == "first"
    error_message = "Expected first element name to be 'first'"
  }

  assert {
    condition     = data.test_data_source.datasource.list_value[0].value == "one"
    error_message = "Expected first element value to be 'one'"
  }

  assert {
    condition     = data.test_data_source.datasource.list_value[1].name == "second"
    error_message = "Expected second element name to be 'second'"
  }

  assert {
    condition     = data.test_data_source.datasource.list_value[1].value == "two"
    error_message = "Expected second element value to be 'two'"
  }
}

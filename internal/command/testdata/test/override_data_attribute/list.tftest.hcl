provider "test" {}

override_data {
  target = data.test_complex_data_source.datasource
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
    condition     = length(data.test_complex_data_source.datasource.list_value) == 2
    error_message = "Expected list_value to have 2 elements, got ${length(data.test_complex_data_source.datasource.list_value)}"
  }

  assert {
    condition     = data.test_complex_data_source.datasource.list_value[0].name == "first"
    error_message = "Expected first element name to be 'first'"
  }
}

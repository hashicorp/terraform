provider "test" {}

override_data {
  target = data.test_complex_data_source.datasource
  values = {
    set_value = [
      {
        name = "first"
      },
      {
        value = "two"
      },
    ]
  }
}

run "test_override_data_complex_set_attribute_partial_elements" {
  command = plan

  assert {
    condition     = length(data.test_complex_data_source.datasource.set_value) == 2
    error_message = "Expected set_value to have 2 elements, got ${length(data.test_complex_data_source.datasource.set_value)}"
  }

  assert {
    condition = length([
      for item in data.test_complex_data_source.datasource.set_value : item
      if item.name == "first" && item.value != null
    ]) == 1
    error_message = "Expected one set_value element with name 'first' and a filled-in value"
  }

  assert {
    condition = length([
      for item in data.test_complex_data_source.datasource.set_value : item
      if item.value == "two" && item.name != null
    ]) == 1
    error_message = "Expected one set_value element with value 'two' and a filled-in name"
  }
}

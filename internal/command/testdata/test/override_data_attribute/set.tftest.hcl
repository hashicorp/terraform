provider "test" {}

override_data {
  target = data.test_complex_data_source.datasource
  values = {
    set_value = [
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

run "test_override_data_set_attribute" {
  command = plan

  assert {
    condition     = length(data.test_complex_data_source.datasource.set_value) == 2
    error_message = "Expected set_value to have 2 elements, got ${length(data.test_complex_data_source.datasource.set_value)}"
  }

  assert {
    condition     = contains([for item in data.test_complex_data_source.datasource.set_value : item.name], "first")
    error_message = "Expected set_value to contain an element with name 'first'"
  }
}

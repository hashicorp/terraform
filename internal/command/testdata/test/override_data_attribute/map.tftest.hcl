provider "test" {}

override_data {
  target = data.test_complex_data_source.datasource
  values = {
    map_value = {
      "key1" = {
        name  = "first"
        value = "one"
      }
      "key2" = {
        name  = "second"
        value = "two"
      }
    }
  }
}

run "test_override_data_map_attribute" {
  command = plan

  assert {
    condition     = length(data.test_complex_data_source.datasource.map_value) == 2
    error_message = "Expected map_value to have 2 elements, got ${length(data.test_complex_data_source.datasource.map_value)}"
  }

  assert {
    condition     = data.test_complex_data_source.datasource.map_value["key1"].name == "first"
    error_message = "Expected key1 name to be 'first'"
  }
}

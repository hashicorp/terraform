
mock_provider "test" {
  mock_resource "test_resource" {
    defaults = {
      id = "resource"
    }
  }

  mock_data "test_data_source" {
    defaults = {
      value = "hello"
    }
  }
}

run "test" {
  variables {
    input = "input"
  }

  assert {
    condition = data.test_data_source.datasource.value == "hello"
    error_message = "wrong value"
  }

  assert {
    condition = test_resource.resource.value == "hello"
    error_message = "wrong value"
  }

  assert {
    condition = test_resource.resource.id == "resource"
    error_message = "wrong value"
  }

}


provider "test" {}

override_resource {
  target = test_resource.resource
  values = {
    id = "resource"
  }
}

override_data {
  target = data.test_data_source.datasource
  values = {
    value = "hello"
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

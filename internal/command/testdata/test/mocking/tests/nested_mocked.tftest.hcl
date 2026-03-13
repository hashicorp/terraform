mock_provider "test" {
  alias = "primary"

  mock_resource "test_complex_resource" {
    defaults = {
      id = "aaaa"
      list_value = [{name  = "first"}, {name  = "second"}]
      nested_list_value = [{name  = "first"}, {name  = "second"}]
      set_value = [{name  = "first"}, {name  = "second"}]
      nested_set_value = [{name  = "first"}, {name  = "second"}]
      map_value = {
        "key1": {
          name  = "first"
        },
        "key2": {
          name  = "third"
        }
      }
      nested_map_value = {
        "key1": {
          name  = "first"
        },
        "key2": {
          name  = "third"
        }
      }
    }
  }
}

variables {
  instances = 1
  child_instances = 0
}


run "test" {

  assert {
    condition = test_complex_resource.primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }
  
  assert {
    condition = test_complex_resource.primary[0].list_value[0].name == "first"
    error_message = "did not apply mocks"
  }
  
  assert {
    condition = test_complex_resource.primary[0].nested_list_value[0].name == "first"
    error_message = "did not apply mocks"
  }
  
  assert {
    condition = test_complex_resource.primary[0].map_value["key1"].name == "first"
    error_message = "did not apply mocks"
  }
  
  assert {
    condition = test_complex_resource.primary[0].nested_map_value["key1"].name == "first"
    error_message = "did not apply mocks"
  }
}

mock_provider "test" {
  alias = "primary"

  mock_resource "test_resource" {
    defaults = {
      id = "aaaa"
      list_value = [{name  = "first"}, {name  = "second"}]
      nested_list_value = [{name  = "first"}, {name  = "second"}]
      set_value = [{name  = "first"}, {name  = "second"}]
      nested_set_value = [{name  = "first"}, {name  = "second"}]
      map_value = [{name  = "first"}, {name  = "second"}]
      nested_map_value = [{name  = "first"}, {name  = "second"}]
    }
  }
}

variables {
  instances = 1
  child_instances = 0
}


run "test" {

  assert {
    condition = test_resource.primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }
}

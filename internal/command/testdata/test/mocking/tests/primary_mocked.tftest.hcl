mock_provider "test" {
  alias = "primary"

  mock_resource "test_resource" {
    defaults = {
      id = "aaaa"
    }
  }
}

variables {
  instances = 1
  child_instances = 1
}


run "test" {

  assert {
    condition = test_resource.primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }

  assert {
    condition = module.child[0].primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }

  assert {
    condition = test_resource.secondary[0].id != "aaaa"
    error_message = "wrongly applied mocks"
  }

  assert {
    condition = module.child[0].secondary[0].id != "aaaa"
    error_message = "wrongly applied mocks"
  }

}

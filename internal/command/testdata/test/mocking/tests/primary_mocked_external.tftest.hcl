mock_provider "test" {
  alias = "primary"

  source ="./tests/mocks"
}

mock_provider "test" {
  alias = "secondary"

  mock_resource "test_resource" {
    override_during = plan
    defaults = {
      id = "bbbb"
    }
  }
}

variables {
  instances = 1
  child_instances = 1
}


run "test" {
  command = plan

  assert {
    condition = test_resource.primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }

  assert {
    condition = module.child[0].primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }

  assert {
    condition = test_resource.secondary[0].id == "bbbb"
    error_message = "wrongly applied mocks"
  }

  assert {
    condition = module.child[0].secondary[0].id == "bbbb"
    error_message = "wrongly applied mocks"
  }

}

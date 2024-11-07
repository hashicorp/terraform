mock_provider "test" {
  alias = "primary"

  mock_resource "test_resource" {
    defaults = {
      id = "aaaa"
    }
  }

  override_resource {
    target = test_resource.primary
    values = {
      id = "bbbb"
    }
  }
}

variables {
  instances = 3
  child_instances = 1
}

run "test" {

  override_resource {
    target = test_resource.primary[1]
    values = {
      id = "cccc"
    }
  }

  assert {
    condition = test_resource.primary[0].id == "bbbb"
    error_message = "did not apply mocks"
  }

  assert {
    condition = test_resource.primary[1].id == "cccc"
    error_message = "did not apply mocks"
  }

  assert {
    condition = test_resource.primary[2].id == "bbbb"
    error_message = "did not apply mocks"
  }

  assert {
    condition = module.child[0].primary[0].id == "aaaa"
    error_message = "did not apply mocks"
  }

}

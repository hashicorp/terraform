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
  instances = 1
  child_instances = 1
}

// This test will fail because the plan command does not use the
// overridden values for computed properties, 
// making the left-hand side of the condition unknown.
run "test" {
  command = plan

  assert {
    condition = test_resource.primary[0].id == "bbbb"
    error_message = "plan should not have the overridden value"
  }

}

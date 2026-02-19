mock_provider "test" {
  alias = "primary"

  mock_resource "test_resource" {
    defaults = {
      id = "aaaa"
    }
  }

  override_resource {
    target = test_resource.primary
    override_during = plan
    values = {
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
    condition = test_resource.primary[0].id == "bbbb"
    error_message = "plan should override the value when override_during is plan"
  }

}

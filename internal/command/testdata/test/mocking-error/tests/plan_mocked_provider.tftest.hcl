mock_provider "test" {
  alias = "secondary"

  mock_resource "test_resource" {
    defaults = {
      id = "ffff"
    }
  }
}


variables {
  instances = 2
  child_instances = 1
}

run "test" {
  command = plan

  assert {
    condition = test_resource.secondary[0].id == "ffff"
    error_message = "plan should use the mocked provider value when override_during is plan"
  }

}

mock_provider "test" {
  alias = "primary"
  override_during = plan

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

  override_resource {
    target = test_resource.primary[1]
    override_during = apply // this should take precedence over the provider-level override_during
    values = {
      id = "bbbb"
    }
  }
}


override_resource {
  target = test_resource.secondary[0]
  override_during = plan
  values = {
    id = "ssss"
  }
}


variables {
  instances = 2
  child_instances = 1
}

run "test" {
  command = plan

  assert {
    condition = test_resource.primary[0].id == "bbbb"
    error_message = "plan should override the value when override_during is plan"
  }

  assert {
    condition = test_resource.secondary[0].id == "ssss"
    error_message = "plan should override the value when override_during is plan"
  }

}

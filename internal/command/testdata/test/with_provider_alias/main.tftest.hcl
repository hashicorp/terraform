provider "test" {
  data_prefix = "data"
  resource_prefix = "resource"
}

provider "test" {
  alias = "setup"

  # The setup provider will write into the main providers data sources.
  resource_prefix = "data"
}

variables {
  managed_id = "B853C121"
}

run "setup" {
  module {
    source = "./setup"
  }

  variables {
    value = "Hello, world!"
    id = "B853C121"
  }

  providers = {
    test = test.setup
  }
}

run "test" {
  assert {
    condition = test_resource.created.value == "Hello, world!"
    error_message = "bad value"
  }
}

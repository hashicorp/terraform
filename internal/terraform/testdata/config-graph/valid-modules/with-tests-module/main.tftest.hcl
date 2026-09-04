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
}

run "test" {
  assert {
    condition = test_resource.created.value == "Hello, world!"
    error_message = "bad value"
  }
}

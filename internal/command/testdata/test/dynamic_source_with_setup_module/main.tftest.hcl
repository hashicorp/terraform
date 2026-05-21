variables {
  managed_id = "B853C121"
}

run "setup" {
  module {
    source = "./setup"
  }

  variables {
    value = "Hello, world!"
    id    = "B853C121"
  }
}

run "test" {
  assert {
    condition     = module.mod.value == "Hello, world!"
    error_message = "expected value from setup module via dynamic source"
  }
}

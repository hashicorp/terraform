variables {
  value = "Hello, world!"
}

run "load_module" {
  module {
    source = "./setup"
  }

  assert {
    condition     = output.value == "Hello, world!"
    error_message = "invalid value"
  }
}
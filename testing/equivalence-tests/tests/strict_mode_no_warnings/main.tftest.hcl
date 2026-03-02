run "test" {
  command = plan

  variables {
    input = "Hello, world!"
  }

  assert {
    condition     = tfcoremock_simple_resource.resource.string == "Hello, world!"
    error_message = "expected string to be Hello, world!"
  }
}

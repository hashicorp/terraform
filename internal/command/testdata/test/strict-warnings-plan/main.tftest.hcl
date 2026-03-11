run "test" {
  command = plan

  variables {
    input      = "Hello, world!"
    undeclared = "this triggers a warning"
  }

  assert {
    condition     = test_resource.resource.value == "Hello, world!"
    error_message = "bad value"
  }
}

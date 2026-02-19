run "validate_test_resource" {
  variables {
    input = "bar"
  }

  assert {
    condition = test_resource.foo.value == "bar"
    error_message = "invalid value"
  }
}

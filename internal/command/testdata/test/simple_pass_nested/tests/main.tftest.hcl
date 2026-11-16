run "validate_test_resource" {
  assert {
    condition = test_resource.foo.value == "bar"
    error_message = "invalid value"
  }
}

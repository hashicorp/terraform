run "validate_test_resource" {
  assert {
    condition = test_resource.foo[0].value == "bar"
    error_message = "invalid value"
  }
}

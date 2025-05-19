run "validate_test_resource" {
  assert {
    condition = test_resource.foo.value == "zap"
    error_message = "invalid value"
  }
}

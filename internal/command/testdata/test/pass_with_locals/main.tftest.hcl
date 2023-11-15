run "validate_test_resource" {
  assert {
    condition = local.value == "bar"
    error_message = "invalid value"
  }
}

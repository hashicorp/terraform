run "validate_test_resource" {

  command = plan

  assert {
    condition = test_resource.foo.value == "bar"
    error_message = "invalid value"
  }
}

run "validate_test_resource" {
  assert {
    condition = test_resource.foo.value == "bar"
    error_message = "invalid value"
  }
}

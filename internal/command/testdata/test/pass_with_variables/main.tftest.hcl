variables {
  input = "bar"
}

run "validate_test_resource" {
  assert {
    condition = test_resource.foo.value == "bar"
    error_message = "invalid value"
  }
}

run "apply_test_resource" {
  variables {
    input = "zap"
  }

  assert {
    condition = test_resource.foo.value == "zap"
    error_message = "invalid value"
  }
}

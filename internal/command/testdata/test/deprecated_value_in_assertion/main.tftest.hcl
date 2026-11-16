run "failing_assertion_on_deprecated_value" {
  assert {
    condition     = test_resource.foo.deprecated == "new-value"
    error_message = "expected to fail"
  }
}

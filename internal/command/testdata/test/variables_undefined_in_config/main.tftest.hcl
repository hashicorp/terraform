run "test" {
  variables {
    input = "value"
  }

  assert {
    condition = test_resource.foo.value == "value"
    error_message = "bad value"
  }
}

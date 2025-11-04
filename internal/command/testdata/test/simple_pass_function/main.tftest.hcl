override_resource {
  target = test_resource.foo
  values = {
    id = format("f-%s", "bar")
  }
}

run "validate_test_resource" {
  assert {
    condition = test_resource.foo.id == "f-bar"
    error_message = "invalid value"
  }
}

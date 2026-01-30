mock_provider "test" {
  mock_resource "test_resource" {
    defaults = {
      id = format("f-%s", "foo")
    }
  }
}

override_resource {
  target = test_resource.bar
  values = {
    id = format("%s-%s", uuid(), "bar")
  }
}

run "validate_test_resource_foo" {
  assert {
    condition = test_resource.foo.id == "f-foo"
    error_message = "invalid value"
  }
}

run "validate_test_resource_bar" {
  assert {
    condition = length(test_resource.bar.id) > 10
    error_message = "invalid value"
  }
}

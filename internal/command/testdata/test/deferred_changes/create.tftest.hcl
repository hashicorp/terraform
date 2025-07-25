
run "create" {
  variables {
    defer = true
  }

  assert {
    condition = test_resource.resource.defer
    error_message = "deferred resource attribute should be true"
  }
}

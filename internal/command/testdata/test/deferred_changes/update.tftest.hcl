
run "create" {
  variables {
    defer = false
  }

  assert {
    condition = !test_resource.resource.defer
    error_message = "deferred resource attribute should be false"
  }
}

run "update" {
  variables {
    defer = true
  }

  assert {
    condition = test_resource.resource.defer
    error_message = "deferred resource attribute should be true"
  }
}

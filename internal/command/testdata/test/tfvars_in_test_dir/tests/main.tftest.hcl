run "primary" {
  assert {
    condition     = var.foo == var.test_foo
    error_message = "Expected: ${var.test_foo}, Actual: ${var.foo}"
  }
}

run "secondary" {
    assert {
      condition     = var.fooJSON == var.test_foo_json
      error_message = "Expected: ${var.test_foo_json}, Actual: ${var.fooJSON}"
    }
}
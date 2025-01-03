run "failing_assertion" {
  assert {
    condition     = local.number < 0
    error_message = "local variable 'number' has a value greater than zero, so this assertion will fail"
  }
}

run "passing_assertion" {
  assert {
    condition     = local.number > 0
    error_message = "local variable 'number' has a value greater than zero, so this assertion will pass"
  }
}

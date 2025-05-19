run "failing_assertion" {
  assert {
    condition     = local.number == 10
    error_message = "assertion 1 should pass"
  }
  assert {
    condition     = local.number < 0
    error_message = "local variable 'number' has a value greater than zero, so assertion 2 will fail"
  }
  assert {
    condition     = local.number == 10
    error_message = "assertion 3 should pass"
  }
}

run "passing_assertion" {
  assert {
    condition     = local.number > 0
    error_message = "local variable 'number' has a value greater than zero, so this assertion will pass"
  }
}

run "should_not_reach" {
  assert {
    condition     = true
    error_message = "should not reach this point"
  }
}

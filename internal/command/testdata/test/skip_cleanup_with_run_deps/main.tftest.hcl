run "test" {
  variables {
    id = "test"
    unused = "unused"
  }
}

run "test_two" {
  state_key = "state"
  skip_cleanup = true
  variables {
    id = "test_two"
    // The output state data for this dependency will also be left behind, but the actual
    // resource will have been destroyed by the cleanup step of test_three.
    unused = run.test.unused
  }
}

run "test_three" {
  variables {
    id = "test_three"
  }
}
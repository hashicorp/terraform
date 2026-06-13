test {
  parallel = true
}

run "test" {
  variables {
    id = "test"
    unused = "unused"
  }
}

run "test_two" {
  state_key = "state2"
  variables {
    // This dependency is a later run, but that should be fine because we are in parallel mode.
    id = run.test_three.id

    // The output state data for this dependency will also be left behind, but the actual
    // resource will have been destroyed by the cleanup step of test_three.
    unused = run.test.unused
  }
}

run "test_three" {
  state_key = "state3"
  variables {
    id = "test_three"
    unused = run.test.unused
  }
}

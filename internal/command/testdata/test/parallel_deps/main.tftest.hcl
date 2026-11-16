test {
  parallel = true
}

run "test_one_a" {
  state_key = "state1"
  variables {
    id = "test_one"
    unused = "unused"
  }
}

run "test_one_b" {
  state_key = "state1"
  variables {
    id = "test_one"
    // even though this references test_three, test_three already has a state
    // dependency via test_one_b, so test_three's state will be destroyed before this run's
    unused = run.test_three.unused  
  }
}

run "test_two" {
  state_key = "state2"
  variables {
    // This dependency is a later run, but that should be fine because we are in parallel mode.
    id = run.test_three.id
  }
}

run "test_three" {
  state_key = "state3"
  variables {
    id = "test_three"
    unused = run.test_one_a.unused
  }
}

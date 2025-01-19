// To run in parallel, sequential runs must have different state keys, and not depend on each other
// NotDepends: true
// DiffStateKey: true

variables {
  foo = "foo"
}


run "setup" {
  parallel = true
  state_key = "start"
  module {
    source = "./setup"
  }

  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "bad"
  }
}

// Depends on previous run, but has different state key, so would not run in parallel
// NotDepends: false
// DiffStateKey: true
run "test_a" {
  parallel = true
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }

  assert {
    condition = run.setup.value == var.foo
    error_message = "triple bad"
  }
}

// Depends on previous run, and has same state key, so would not run in parallel
// NotDepends: false
// DiffStateKey: false
run "test_b" {
  parallel = true
  variables {
    input = run.test_a.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }

  assert {
    condition = run.setup.value == var.foo
    error_message = "triple bad"
  }
}

// Does not depend on previous run, and has same state key, so would not run in parallel
// NotDepends: true
// DiffStateKey: false
run "test_c" {
  parallel = true
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

// Does not depend on previous run, and has different state key, so would run in parallel
// NotDepends: true
// DiffStateKey: true
// However, it has a state key that is the same as run.setup, so it should wait for that run, and
// thus may run in parallel with test_a
run "test_d" {
  parallel = true
  state_key = "start"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

// Generate additional runs
// NotDepends: true
// DiffStateKey: true
run "test_1" {
  parallel = true
  state_key = "state_foo"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_1"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_2" {
  parallel = true
  state_key = "state_bar"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_2"
  }
}

// NotDepends: true
// DiffStateKey: false
run "test_3" {
  parallel = true
  state_key = "start"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_3"
  }
}

// NotDepends: false
// DiffStateKey: false
run "test_4" {
  parallel = true
  state_key = "start"
  variables {
    input = run.test_2.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_4"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_5" {
  parallel = true
  state_key = "state_baz"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_5"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_6" {
  parallel = true
  state_key = "state_qux"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_6"
  }
}

// NotDepends: true
// DiffStateKey: false
run "test_7" {
  parallel = true
  state_key = "start"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_7"
  }
}

// NotDepends: false
// DiffStateKey: false
run "test_8" {
  parallel = true
  state_key = "start"
  variables {
    input = run.test_6.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_8"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_9" {
  parallel = true
  state_key = "state_foo"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_9"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_10" {
  parallel = true
  state_key = "state_bar"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_10"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_11" {
  parallel = true
  state_key = "state_foo"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_11"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_12" {
  parallel = true
  state_key = "state_bar"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_12"
  }
}

// NotDepends: true
// DiffStateKey: false
run "test_13" {
  parallel = true
  state_key = "start"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_13"
  }
}

// NotDepends: false
// DiffStateKey: false
run "test_14" {
  parallel = true
  state_key = "start"
  variables {
    input = run.test_12.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_14"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_15" {
  parallel = true
  state_key = "state_baz"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_15"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_16" {
  parallel = true
  state_key = "state_qux"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_16"
  }
}

// NotDepends: true
// DiffStateKey: false
run "test_17" {
  parallel = true
  state_key = "start"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_17"
  }
}

// NotDepends: false
// DiffStateKey: false
run "test_18" {
  parallel = true
  state_key = "start"
  variables {
    input = run.test_16.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_18"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_19" {
  parallel = true
  state_key = "state_foo"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_19"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_20" {
  parallel = true
  state_key = "state_bar"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_20"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_21" {
  parallel = true
  state_key = "state_foo"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_21"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_22" {
  parallel = true
  state_key = "state_bar"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_22"
  }
}

// NotDepends: true
// DiffStateKey: false
run "test_23" {
  parallel = true
  state_key = "start"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_23"
  }
}

// NotDepends: false
// DiffStateKey: false
run "test_24" {
  parallel = true
  state_key = "start"
  variables {
    input = run.test_22.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_24"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_25" {
  parallel = true
  state_key = "state_baz"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_25"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_26" {
  parallel = true
  state_key = "state_qux"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_26"
  }
}

// NotDepends: true
// DiffStateKey: false
run "test_27" {
  parallel = true
  state_key = "start"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_27"
  }
}

// NotDepends: false
// DiffStateKey: false
run "test_28" {
  parallel = true
  state_key = "start"
  variables {
    input = run.test_26.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_28"
  }
}

// NotDepends: true
// DiffStateKey: true
run "test_29" {
  parallel = true
  state_key = "state_foo"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_29"
  }
}

// NotDepends: false
// DiffStateKey: true
run "test_30" {
  parallel = true
  state_key = "state_bar"
  variables {
    input = run.setup.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "error in test_30"
  }
}

// Expected order:
//   - run [setup]
//   - run [test_a, test_d]
//   - run [test_b]
//   - run [test_c]


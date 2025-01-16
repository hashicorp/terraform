// To run in parallel, sequential runs must have different state keys, and not depend on each other
// NotDepends: true
// DiffStateKey: true

variables {
  foo = "foo"
}


run "setup" {
  parallel = true
  state_key = "test_d"
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
// However, it has a state key that is the same as a previous run, so it should wait for that run.
run "test_d" {
  parallel = true
  state_key = "test_d"
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
# run "test_d" {
#   parallel = true
#   state_key = "test_d"
#   variables {
#     input = "foo"
#   }

#   assert {
#     condition = output.value == var.foo
#     error_message = "double bad"
#   }
# }

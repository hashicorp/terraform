// To run in parallel, sequential runs must have different state keys, and not depend on each other
// NotDepends: true
// DiffStateKey: true
test {
  // This would set the parallel flag to true in all runs
  parallel = true
}

variables {
  foo = "foo"
}

provider "test" {
}

provider "test" {
  alias = "start"
}

run "test_a" {
  state_key = "state_foo"
  variables {
    input = "foo"
  }
  providers = {
    test = test
  }

  assert {
    condition     = output.value == var.foo
    error_message = "error in test_a"
  }
}

run "test_b" {
  state_key = "state_bar"
  variables {
    input = "bar"
  }

  providers = {
    test = test.start
  }

  assert {
    condition     = output.value == "bar"
    error_message = "error in test_b"
  }
}

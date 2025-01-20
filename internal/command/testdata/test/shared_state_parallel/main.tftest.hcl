test {
  // This would set the parallel flag to true in all runs
  parallel = true
}

variables {
  foo = "foo"
}


run "main_first" {
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

run "main_second" {
  variables {
    input = run.main_first.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }

  assert {
    condition = run.main_first.value == var.foo
    error_message = "triple bad"
  }
}

run "main_third" {
  variables {
    input = run.main_second.value
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }

  assert {
    condition = run.main_first.value == var.foo
    error_message = "triple bad"
  }
}

run "main_fourth" {
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

// The satisfies all the conditions to run in parallel, but the parallel flag is set to false,
// so it should run in sequence
run "main_fifth" {
  state_key = "start"
  parallel = false
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

// Expected order:
//   - run [main_first]
//   - run [main_second]
//   - run [main_third]
//   - run [main_fourth]
//   - run [main_fifth]


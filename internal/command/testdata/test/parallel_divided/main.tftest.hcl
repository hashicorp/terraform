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
  state_key = "uniq_3"
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
  parallel = false // effectively dividing the parallelizable group into two
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

// Because of the division caused by main_fourth, main_fifth and main_sixth would run in parallel,
// but would only run after main_fourth has completed.
run "main_fifth" {
  state_key = "uniq_5"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

run "main_sixth" {
  state_key = "uniq_6"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}
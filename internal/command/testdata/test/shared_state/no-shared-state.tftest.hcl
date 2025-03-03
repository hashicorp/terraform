variables {
  foo = "foo"
}


run "setup" {
  state_key = "setup"
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

run "test_a" {
  state_key = "test_a"
  variables {
    input = "foo"
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

run "test_b" {
  state_key = "test_b"
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

run "test_c" {
  state_key = "test_c"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}


run "test_d" {
  state_key = "test_d"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

run "test_e" {
  state_key = "test_e"
  variables {
    input = "foo"
  }

  assert {
    condition = output.value == var.foo
    error_message = "double bad"
  }
}

variables {
  foo = "foo"
}


run "setup" {
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

run "test" {

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

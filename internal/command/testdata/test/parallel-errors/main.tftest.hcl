test {
  parallel = true
}

run "one" {
  state_key = "one"

  variables {
    input = "one"
  }

  assert {
    condition = output.value
    error_message = "something"
  }
}

run "two" {
  state_key = "two"

  variables {
    input = run.one.value
  }
}

run "three" {
  state_key = "three"

  variables {
    input = "three"
  }
}

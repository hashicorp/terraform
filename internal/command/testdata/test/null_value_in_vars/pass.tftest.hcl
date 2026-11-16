
run "first" {
  variables {
    interesting_input = "bar"
  }
}

run "second" {
  variables {
    interesting_input = "bar"
    null_input = run.first.null_output
  }

  assert {
    condition     = output.null_output == run.first.null_output
    error_message = "should have passed"
  }
}

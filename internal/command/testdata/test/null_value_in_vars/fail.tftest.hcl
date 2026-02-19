
run "first" {
  variables {
    interesting_input = "bar"
  }
}

run "second" {
  variables {
    // It shouldn't let this happen.
    interesting_input = run.first.null_output
  }
}

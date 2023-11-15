variables {
  input = "default"
}

run "test_run_one" {
  expect_failures = [
    input.input,
    output.output,
  ]
}

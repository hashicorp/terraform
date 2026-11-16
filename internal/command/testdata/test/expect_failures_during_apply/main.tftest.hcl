run "test" {

  command = apply

// We are expecting the output to fail during apply, but it will not, so the test will fail.
  expect_failures = [
    output.output
  ]
}

// this should still run
run "follow-up" {
  command = apply

  variables {
    input = "does not matter"
  }
}
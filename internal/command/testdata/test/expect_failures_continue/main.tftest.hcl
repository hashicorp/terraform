variables {
  input = "some value"
}

run "test" {

  command = apply

  expect_failures = [
    output.output
  ]
}

run "follow-up" {
  command = apply

  variables {
    input = "something incredibly specific"
  }
}
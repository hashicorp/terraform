variables {
  input = "some value"
}

run "test" {
  expect_failures = [
    check.expected_to_fail
  ]
}

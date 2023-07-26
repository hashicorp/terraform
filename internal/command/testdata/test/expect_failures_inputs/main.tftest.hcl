variables {
  input = "some value"
}

run "test" {
  expect_failures = [
    var.input
  ]
}

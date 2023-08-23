variables {
  input = "some value"
}

run "test" {
  command = plan

  expect_failures = [
    var.input
  ]
}

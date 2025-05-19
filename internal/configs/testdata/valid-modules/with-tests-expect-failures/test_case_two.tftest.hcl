variables {
  input = "default"
}

run "test_run_one" {
  expect_failures = [
    foo_resource.a,
  ]
}

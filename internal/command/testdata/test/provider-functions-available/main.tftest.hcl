
run "test" {
  assert {
    condition = provider::test::is_true(output.value)
    error_message = "bad response"
  }
}

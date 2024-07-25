run "empty" {
  assert {
    condition = module.empty.value == "Hello, World!"
    error_message = "wrong output value"
  }
}

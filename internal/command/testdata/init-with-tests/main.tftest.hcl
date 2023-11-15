run "test" {
  assert {
    condition = test_instance.foo.ami == "bar"
    error_message = "incorrect value"
  }
}

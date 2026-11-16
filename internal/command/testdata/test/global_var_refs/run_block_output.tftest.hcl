
variables {
  input = var.setup.value
}

run "setup" {
  variables {
    input = "hello"
  }
}

run "execute" {
  assert {
    condition     = output.value == "hello"
    error_message = "bad output value"
  }
}

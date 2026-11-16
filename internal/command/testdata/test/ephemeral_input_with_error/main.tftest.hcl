run "validate_ephemeral_input" {
  variables {
    foo = "baz"
  }
  assert {
    condition = var.foo == "bar"
    error_message = "Expecting this to fail, real value is: ${var.foo}"
  }
}

run "validate_ephemeral_input_is_ephemeral" {
  variables {
    foo = "bar"
  }
  assert {
    condition = ephemeralasnull(var.foo) == null
    error_message = "Should be ephemeral"
  }
}

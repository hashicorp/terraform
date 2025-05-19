run "validate_ephemeral_input" {
  variables {
    foo = "bar"
  }
  assert {
    condition = var.foo == "bar"
    error_message = "Should be accessible"
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

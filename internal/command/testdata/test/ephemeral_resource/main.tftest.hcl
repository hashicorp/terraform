run "validate_ephemeral_resource" {
  assert {
    condition = ephemeral.test_ephemeral_resource.data.value == "bar"
    error_message = "We expect this to fail since ephemeral resources should be closed when this is evaluated"
  }
}

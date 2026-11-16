# Test that override_ephemeral within a mock_provider block overrides the
# mock_ephemeral default for a specific ephemeral resource instance.
# The overridden value is surfaced via an ephemeral output.

mock_provider "test" {
  mock_ephemeral "test_ephemeral_resource" {
    defaults = {
      value = "mocked_value"
    }
  }

  override_ephemeral {
    target = ephemeral.test_ephemeral_resource.data
    values = {
      value = "overridden_value"
    }
  }
}

run "validate_override_ephemeral" {
  assert {
    condition     = output.ephemeral_value == "overridden_value"
    error_message = "Expected override_ephemeral to take precedence over mock_ephemeral default"
  }
}

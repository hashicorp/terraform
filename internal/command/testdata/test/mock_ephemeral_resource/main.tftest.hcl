# Test that mock_ephemeral within a mock_provider block provides default values
# for ephemeral resources. The mocked value is surfaced via an ephemeral output.

mock_provider "test" {
  mock_ephemeral "test_ephemeral_resource" {
    defaults = {
      value = "mocked_value"
    }
  }
}

run "validate_mock_ephemeral" {
  assert {
    condition     = output.ephemeral_value == "mocked_value"
    error_message = "Expected mock_ephemeral default value to be returned by the mocked provider"
  }
}

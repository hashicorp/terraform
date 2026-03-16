
mock_provider "test" {
  source = "./testing/base"

  mock_resource "test_resource" {
    defaults = {
      id = "local-mock-id" // should override the file-based mock
    }
  }
}

mock_provider "test" {
  source = "./testing/base"
  alias = "secondary"
}

run "test_foo" {
  assert {
    condition     = output.foo == "local-mock-id"
    error_message = "invalid value"
  }
}


run "test_bar" {
  assert {
    condition     = output.bar == "file-mock-id"
    error_message = "invalid value"
  }
}

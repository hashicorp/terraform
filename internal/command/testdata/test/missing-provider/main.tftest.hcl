
provider "test" {}

run "passes_validation" {
  assert {
    condition = test_resource.primary.value == "foo"
    error_message = "primary contains invalid value"
  }

  assert {
    condition = test_resource.secondary.value == "bar"
    error_message = "secondary contains invalid value"
  }
}

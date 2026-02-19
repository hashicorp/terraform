
provider "test" {}

provider "test" {
  alias = "secondary"
}

run "passes_validation_primary" {

  providers = {
    test = test
  }

  assert {
    condition = test_resource.primary.value == "foo"
    error_message = "primary contains invalid value"
  }

}

run "passes_validation_secondary" {

  providers = {
    test = test
  }

  module {
    source = "./setup"
  }

  assert {
    condition = test_resource.primary.value == "foo"
    error_message = "primary contains invalid value"
  }

  assert {
    condition = test_resource.secondary.value == "bar"
    error_message = "secondary contains invalid value"
  }
}
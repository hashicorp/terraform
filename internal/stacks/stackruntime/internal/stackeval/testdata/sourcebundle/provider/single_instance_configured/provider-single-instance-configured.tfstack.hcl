# Set the test-only global "provider_configuration" to a value that should
# be assigned to the "test" argument in the provider configuration.

required_providers {
  foo = {
    source = "terraform.io/builtin/foo"
  }
}

provider "foo" "bar" {
  config {
    test = _test_only_global.provider_configuration
  }
}

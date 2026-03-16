# Set the test-only global "provider_instances" to the value that should be
# assigned to the provider blocks' for_each argument.

required_providers {
  foo = {
    source = "terraform.io/builtin/foo"
  }
}

provider "foo" "bar" {
  for_each = _test_only_global.provider_instances
}

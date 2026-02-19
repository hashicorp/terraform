terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {
      region = "saturn"
    }
    value = "foobar"
  }
}

# This config is valid, but the test will force the provider
# or state store's config validation methods to return an error.

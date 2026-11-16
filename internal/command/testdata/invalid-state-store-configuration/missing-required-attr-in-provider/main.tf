terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {
      # Test mock provider will create a required attribute for the provider
      # and there are no attributes here in the config...
    }
    value = "foobar"
  }
}

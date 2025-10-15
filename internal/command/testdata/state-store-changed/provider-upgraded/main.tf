terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
      # version = "9.9.9" // We've now specified using v9.9.9, versus the v1.2.3 used at last init and in the backend state file
    }
  }
  state_store "test_store" {
    provider "test" {
      region = "foobar"
    }

    value = "foobar"
  }
}

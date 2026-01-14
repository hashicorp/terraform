terraform {
  experiments = [pluggable_state_stores]
  required_providers {
    test = {
      source = "hashicorp/test"
      # No version constraints here; we assume the test using this fixture forces the latest provider version
      # to not match the backend state file in this folder.
    }
  }
  state_store "test_store" {
    provider "test" {
      region = "foobar"
    }

    value = "foobar"
  }
}

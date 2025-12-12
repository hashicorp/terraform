terraform {
  # There should be `experiments = [pluggable_state_stores]` present here, but it is intentionally missing.
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {
    }

    value = "foobar"
  }
}

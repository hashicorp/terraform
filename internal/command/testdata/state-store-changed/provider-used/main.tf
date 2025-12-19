terraform {
  experiments = [pluggable_state_stores]
  required_providers {
    test2 = {
      source = "hashicorp/test2"
    }
  }

  # changed to using `test2` provider, versus `test` used in the backend state file
  state_store "test2_store" {
    provider "test2" {
      region = "foobar"
    }

    value = "foobar"
  }
}

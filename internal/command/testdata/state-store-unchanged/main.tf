terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}

    value = "foobar" # matches backend state file
  }
}

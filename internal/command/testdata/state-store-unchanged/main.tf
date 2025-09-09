terraform {
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "1.2.3"
    }
  }
  state_store "test_store" {
    provider "test" {}

    value = "foobar" # matches backend state file
  }
}

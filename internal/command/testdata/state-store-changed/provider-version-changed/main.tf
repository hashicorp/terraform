terraform {
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "~> 1.0"
    }
  }
  state_store "test_store" {
    provider "test" {
      region = "foobar"
    }

    value = "foobar"
  }
}

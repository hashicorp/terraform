terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_nonexistent" { # nonexistent is not a valid state store type in the mocked provider
    provider "test" {}
  }
}

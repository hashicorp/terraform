terraform {
  backend "foo" {}

  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}

    value = "override"
  }
}

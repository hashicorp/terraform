terraform {
  required_providers {
    test = {
      source = "registry.terraform.io/hashicorp/test"
    }
  }

  state_store "test_store" {
    provider "test" {}

    value = "foobar"
  }
}

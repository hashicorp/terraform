terraform {
  cloud {
    organization = "foo"
    workspaces {
      name = "value"
    }
  }

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

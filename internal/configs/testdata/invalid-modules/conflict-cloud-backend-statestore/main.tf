terraform {
  backend "foo" {}

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

  experiments = [pluggable_state_stores]
  state_store "test_store" {
    provider "test" {}

    value = "override"
  }
}

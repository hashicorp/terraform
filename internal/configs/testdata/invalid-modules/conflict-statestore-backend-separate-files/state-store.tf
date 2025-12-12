terraform {
  experiments = [pluggable_state_stores]
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}

    custom_attr = "override"
  }
}

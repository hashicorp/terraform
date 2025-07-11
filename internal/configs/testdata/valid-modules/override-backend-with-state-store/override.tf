terraform {
  required_providers {
    foo = {
      source = "my-org/foo"
    }
  }
  state_store "foo_override" {
    provider "foo" {}

    custom_attr = "override"
  }
}

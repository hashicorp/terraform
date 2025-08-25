terraform {

  required_providers {
    foo = {
      source = "my-org/foo"
    }
  }
  state_store "foo_foo" {
    provider "foo" {}
  }
}

terraform {
  required_providers {
    foo = {
      source = "my-org/foo"
    }
  }
  state_store "foo_bar" {
    provider "foo" {}

    bar = "foobar"
  }
}

variable "foo" { default = "bar" }

terraform {
  required_providers {
    foo = {
      source = "my-org/foo"
    }
  }
  state_store "foo_bar" {
    provider "foo" {
      region = var.foo
    }

    bar = "hardcoded"
  }
}

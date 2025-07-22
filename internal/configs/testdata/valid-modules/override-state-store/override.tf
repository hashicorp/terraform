terraform {
  required_providers {
    bar = {
      source = "my-org/bar"
    }
  }
  state_store "foo_override" {
    provider "bar" {}

    custom_attr = "override"
  }
}

provider "bar" {}

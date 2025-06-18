terraform {
  backend "foo" {}

  cloud {
    organization = "foo"
    workspaces {
      name = "value"
    }
  }

  state_store "foo_bar" {
    provider "foo" {}

    custom_attr = "override"
  }
}

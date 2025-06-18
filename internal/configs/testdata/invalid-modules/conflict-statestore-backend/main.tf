terraform {
  backend "foo" {}

  state_store "foo_bar" {
    provider "foo" {}

    custom_attr = "override"
  }
}

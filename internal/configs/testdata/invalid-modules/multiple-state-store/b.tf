terraform {
  state_store "bar_bar" {
    provider "bar" {}

    custom_attr = "override"
  }
}

terraform {
  experiments = [pluggable_state_stores]
  state_store "bar_bar" {
    provider "bar" {}

    custom_attr = "override"
  }
}

terraform {
  experiments = [pluggable_state_stores]
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}

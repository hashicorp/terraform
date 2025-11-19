terraform {
  state_store "test_store" {
    provider "test" {}
    value = "foobar"
  }
}

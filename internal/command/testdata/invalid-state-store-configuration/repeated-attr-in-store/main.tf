terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}
    value = "value1"
    value = "value2" # Should trigger an error
  }
}

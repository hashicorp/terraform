terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {
      region = "region1"
      region = "region2" # Should trigger an error
    }
  }
}

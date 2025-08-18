terraform {
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "1.2.3"
    }
  }
  state_store "test_inmem" {
    provider "test" {}

    value = "foobar"
  }
}

terraform {
  experiments = [pluggable_state_stores]
  required_providers {
    test = {
      source  = "hashicorp/test"
      version = "1.2.3"
    }
  }
  state_store "test_store" {
    provider "test" {}

    value = "foobar"
  }
}

resource "test_instance" "foo" {
  ami = "bar"
}

terraform {
  # Not including an experiments list here
  # See https://github.com/hashicorp/terraform/issues/38012
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
  state_store "test_store" {
    provider "test" {}

    value = "foobar"
  }
}

provider "test" {}

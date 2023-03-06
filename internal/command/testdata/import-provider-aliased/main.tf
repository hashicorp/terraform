provider "test" {
    foo = "bar"

    alias = "alias"
}

resource "test_instance" "foo" {
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

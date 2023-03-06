provider "test" {
    foo = "bar"
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

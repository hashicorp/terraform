resource "test_instance" "foo" {
  id = "foo"
}

resource "test_instance" "bar" {
  id = "bar"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

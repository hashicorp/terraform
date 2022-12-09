terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "foo" {
  count = 1
  ami = "bar"
}

data "test_ds" "bar" {
  filter = "foo"
}

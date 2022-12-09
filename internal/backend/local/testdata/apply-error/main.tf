terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

resource "test_instance" "foo" {
    ami = "bar"
}

resource "test_instance" "bar" {
    ami = "error"
}

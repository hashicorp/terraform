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

provider "test" {
    value = "foo"
}

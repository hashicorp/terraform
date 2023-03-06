resource "test_instance" "foo" {
    ami = "bar"
}

resource "test_instance" "bar" {
    ami = "${test_instance.foo.ami}"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}

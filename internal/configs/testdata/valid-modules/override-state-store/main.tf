terraform {
  required_providers {
    foo = {
      source = "my-org/foo"
    }
  }
  state_store "foo_bar" {
    provider "foo" {}

    custom_attr = "foobar"
  }
}

resource "aws_instance" "web" {
  ami = "ami-1234"
  security_groups = [
    "foo",
    "bar",
  ]
}

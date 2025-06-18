terraform {
  // Note: not valid config - a paired entry in required_providers is usually needed
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

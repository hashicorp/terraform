provider "aws" {
    value = test_instance.foo.id
}

resource "aws_instance" "bar" {}

resource "test_instance" "foo" {
    value = "yes"
}

provider "aws" {}

resource "aws_instance" "foo" {
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.id}"
}

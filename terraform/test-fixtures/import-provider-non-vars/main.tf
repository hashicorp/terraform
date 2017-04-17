provider "aws" {
    foo = "${aws_instance.foo.bar}"
}

resource "aws_instance" "foo" {
    bar = "value"
}

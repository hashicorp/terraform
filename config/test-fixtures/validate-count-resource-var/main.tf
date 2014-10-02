resource "aws_instance" "foo" {}

resource "aws_instance" "web" {
    count = "${aws_instance.foo.bar}"
}

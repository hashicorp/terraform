resource "aws_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.num}"
}

resource "aws_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    num = "${aws_instance.foo.num}"
}

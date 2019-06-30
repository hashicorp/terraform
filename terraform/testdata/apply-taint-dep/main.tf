resource "aws_instance" "foo" {
    num = "2"
}

resource "aws_instance" "bar" {
    num = "2"
    foo = "${aws_instance.foo.id}"
}

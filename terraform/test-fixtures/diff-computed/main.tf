resource "aws_instance" "foo" {
    num = "2"
    compute = "id"
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.id}"
}

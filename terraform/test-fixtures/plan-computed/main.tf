resource "aws_instance" "foo" {
    num = "2"
    compute = "foo"
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.foo}"
}

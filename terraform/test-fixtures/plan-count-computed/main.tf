resource "aws_instance" "foo" {
    num = "2"
    compute = "foo"
}

resource "aws_instance" "bar" {
    count = "${aws_instance.foo.foo}"
}

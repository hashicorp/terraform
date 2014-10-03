resource "aws_instance" "foo" {
    count = 0
    foo = "foo"
}

resource "aws_instance" "bar" {
    foo = "${aws_instance.foo.*.foo}"
}
